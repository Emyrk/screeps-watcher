package screepssocket

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Emyrk/screeps-watcher/watch/auth"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"nhooyr.io/websocket"
)

type ScreepsWebsocket struct {
	URL        *url.URL
	logger     zerolog.Logger
	authMethod auth.Method
	cli        *http.Client
	userID     string
	channels   []string

	session *Session
	reg     *prometheus.Registry

	// metrics
	websocketCPU         prometheus.Gauge
	websocketMemoryBytes prometheus.Gauge
}

func New(ctx context.Context, URL *url.URL, logger zerolog.Logger, cli *http.Client, authMethod auth.Method, channels []string, labels prometheus.Labels) (*ScreepsWebsocket, error) {
	wbs := &ScreepsWebsocket{
		URL:        URL,
		logger:     logger,
		authMethod: authMethod,
		cli:        cli,
		channels:   channels,
		reg:        prometheus.NewRegistry(),
		websocketCPU: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   "screeps",
			Subsystem:   "websocket",
			Name:        "cpu_last",
			Help:        "Last recorded cpu usage for the user.",
			ConstLabels: labels,
		}),
		websocketMemoryBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   "screeps",
			Subsystem:   "websocket",
			Name:        "memory_last_bytes",
			Help:        "Last recorded memory usage for the user in bytes.",
			ConstLabels: labels,
		}),
	}

	wbs.reg.MustRegister(wbs.websocketCPU)
	wbs.reg.MustRegister(wbs.websocketMemoryBytes)

	_, err := wbs.newURL()
	if err != nil {
		wbs.logger.Error().Err(err).Msg("Failed to create websocket URL")
		return nil, err
	}
	userID, err := wbs.MyUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user ID: %w", err)
	}
	wbs.userID = userID

	return wbs, nil
}

func (s *ScreepsWebsocket) Collect(ch chan<- prometheus.Metric) {
	s.reg.Collect(ch)
}

func (s *ScreepsWebsocket) Describe(descs chan<- *prometheus.Desc) {
	s.reg.Describe(descs)
}

type userInfo struct {
	ID string `json:"_id"`
}

func (s *ScreepsWebsocket) MyUserID(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.URL.ResolveReference(&url.URL{
		Path: "/api/auth/me",
	}).String(), nil)
	if err != nil {
		return "", fmt.Errorf("make me request: %w", err)
	}

	resp, err := s.authMethod.AuthenticatedRequest(s.cli, req)
	if err != nil {
		return "", fmt.Errorf("do me request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Println(req.URL.String())
	d, _ := io.ReadAll(resp.Body)
	fmt.Println(string(d))

	var uinfo userInfo
	err = json.NewDecoder(resp.Body).Decode(&uinfo)
	if err != nil {
		return "", fmt.Errorf("decode me response: %w", err)
	}

	if uinfo.ID == "" {
		return "", fmt.Errorf("empty user ID")
	}

	return uinfo.ID, nil
}

func (s *ScreepsWebsocket) newURL() (*url.URL, error) {
	chars := make([]byte, 4)
	crand.Read(chars)
	wsURL, err := s.URL.Parse(fmt.Sprintf("/socket/%d/%s/websocket", rand.Intn(899)+100, hex.EncodeToString(chars)))
	if err != nil {
		return nil, fmt.Errorf("make websocket url: %w", err)
	}

	switch s.URL.Scheme {
	case "http":
		wsURL.Scheme = "ws"
	case "https":
		wsURL.Scheme = "wss"
	default:
		return nil, fmt.Errorf("unsupported scheme %s", s.URL.Scheme)
	}

	return wsURL, nil
}

func (s *ScreepsWebsocket) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:

		}

		session, err := s.dial(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to dial websocket, will retry...")
			time.Sleep(time.Second * 10)
			continue
		}

		s.logger.Info().Msg("Websocket session started")
		s.session = session
		err = session.Watch(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Websocket session failed")
			time.Sleep(time.Second * 10)
			continue
		}
	}
}

func (s *ScreepsWebsocket) reportCPU(msg any) {
	msgMsp, ok := msg.(map[string]any)
	if !ok {
		return
	}

	cpu, ok := msgMsp["cpu"]
	if ok {
		cpuNum, ok := cpu.(float64)
		if ok {
			s.websocketCPU.Set(cpuNum)
		}
	}

	memBytes, ok := msgMsp["memory"]
	if ok {
		memBytesNum, ok := memBytes.(float64)
		if ok {
			s.websocketMemoryBytes.Set(memBytesNum)
		}
	}
}

type Session struct {
	websocket *ScreepsWebsocket
	conn      *websocket.Conn
	logger    zerolog.Logger

	subscribeTo map[string]bool
}

func (s *ScreepsWebsocket) channelsMap() map[string]bool {
	m := make(map[string]bool)
	for _, c := range s.channels {
		switch c {
		case "console":
			m[fmt.Sprintf("user:%s/console", s.userID)] = false
		case "cpu":
			m[fmt.Sprintf("user:%s/cpu", s.userID)] = false
		default:
			s.logger.Warn().Str("channel", c).Msg("Unknown channel")
		}
	}
	return m
}

func (s *ScreepsWebsocket) dial(ctx context.Context) (*Session, error) {
	socketURL, err := s.newURL()
	if err != nil {
		return nil, fmt.Errorf("make websocket url: %w", err)
	}

	conn, _, err := websocket.Dial(ctx, socketURL.String(), &websocket.DialOptions{})
	if err != nil {
		return nil, fmt.Errorf("dial websocket %s: %w", socketURL.String(), err)
	}
	if conn == nil {
		return nil, fmt.Errorf("dial websocket %s: nil connection", socketURL.String())
	}

	buf := make([]byte, 3)
	_, _ = crand.Read(buf)
	return &Session{
		websocket:   s,
		conn:        conn,
		logger:      s.logger.With().Str("instance", hex.EncodeToString(buf)).Logger(),
		subscribeTo: s.channelsMap(),
	}, nil
}

func (s *Session) Watch(ctx context.Context) error {
	for {
		_, data, err := s.conn.Read(ctx)
		if websocket.CloseStatus(err) != -1 {
			return fmt.Errorf("websocket closed: %w", err)
		}
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to read from websocket")
		}

		err = s.handleIncomingMessage(ctx, data)
		if err != nil {
			_ = s.Close()
			s.logger.Error().Err(err).Msg("Failed to handle incoming message")
			return err
		}
	}
}

func (s *Session) Close() error {
	return s.conn.Close(websocket.StatusNormalClosure, "")
}

// handleIncomingMessage
// https://gist.github.com/bzy-xyz/9c4d8c9f9498a2d7983d
// https://github.com/daboross/rust-screeps-api/blob/master/protocol-docs/websocket.md#known-channels
func (s *Session) handleIncomingMessage(ctx context.Context, data []byte) error {
	// 1 char prefixes
	if len(data) < 1 {
		return nil
	}

	switch data[0] {
	case 'h':
		// heartbeat, do nothing
		return nil
	case 'a':
		// Data!
		msgs, err := batchPayload(ctx, data[1:])
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to batch payload")
			return nil
		}

		for _, msg := range msgs {
			s.handleMessage(ctx, msg)
		}
	case 'm':
		msg := data[1:]
		s.handleMessage(ctx, msg)
		return nil
	case 'o':
		token, err := s.websocket.authMethod.Token(ctx, s.websocket.URL, s.websocket.cli)
		if err != nil {
			return fmt.Errorf("get token to auth websocket: %w", err)
		}

		err = s.WriteMessage(ctx, fmt.Sprintf("auth %s", token))
		if err != nil {
			return fmt.Errorf("write auth to websocket: %w", err)
		}
	default:
		s.logger.Info().Str("type", string(data[0])).Str("msg", string(data[1:])).Msg("Unknown message type")
	}
	return nil
}

func (s *Session) handleMessage(ctx context.Context, jmsg json.RawMessage) {
	var msg any
	err := json.Unmarshal(jmsg, &msg)
	if err != nil {
		s.logger.Error().Err(err).RawJSON("msg", jmsg).Msg("Failed to unmarshal message")
		return
	}

	switch msg.(type) {
	case string:
		s.handleStringMessage(ctx, msg.(string))
	case []any:
		s.handleSliceMessage(ctx, msg.([]any))
	default:
		s.logger.Info().Type("msg", msg).Msg("Unknown message type")
	}
	return
}

func (s *Session) handleStringMessage(ctx context.Context, message string) {
	switch {
	case strings.HasPrefix(message, "auth ok"):
		for k, v := range s.subscribeTo {
			if v {
				continue
			}

			err := s.WriteMessage(ctx, fmt.Sprintf("subscribe %s", k))
			if err != nil {
				s.logger.Error().Err(err).Msgf("Failed to subscribe to %s", k)
			}
		}
	}
}

var channelRegex = regexp.MustCompile(`^(?P<channel_type>user):(?P<user_id>[a-f0-9]+)/(?P<channel_name>.*)$`)

func (s *Session) handleSliceMessage(ctx context.Context, msg []any) {
	if len(msg) != 2 {
		data, _ := json.Marshal(msg)
		s.logger.Error().Int("len", len(msg)).RawJSON("payload", data).Msg("Unknown slice message")
		return
	}

	switch msg[0].(type) {
	case string:
		matches := channelRegex.FindStringSubmatch(msg[0].(string))
		if matches == nil {
			s.logger.Error().Str("channel", msg[0].(string)).Msg("Failed to match channel")
			return
		}
		channelType := matches[channelRegex.SubexpIndex("channel_type")]
		channelName := matches[channelRegex.SubexpIndex("channel_name")]
		userID := matches[channelRegex.SubexpIndex("user_id")]

		// TODO: handle more here
		if channelType == "user" && channelName == "console" {
			LogConsolePayload(s.logger.With().
				Str("channel_type", channelType).
				Str("channel_name", channelName).
				Str("user_id", userID).
				Logger(), msg[1])
			return
		}

		if channelType == "user" && channelName == "cpu" {
			s.websocket.reportCPU(msg[1])
			return
		}

		s.logger.Error().Str("channel_type", channelType).Str("channel_name", channelName).Msg("Unknown channel")
	default:
		s.logger.Error().Type("msg", msg[0]).Msg("Unknown message type in slice index 0")
	}
}

func (s *Session) WriteMessage(ctx context.Context, message string) error {
	return s.conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf("[%q]", message)))
}
