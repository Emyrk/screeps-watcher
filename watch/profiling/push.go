package profiling

import (
	"bytes"
	"fmt"
	"time"

	"github.com/google/pprof/profile"
	"github.com/grafana/pyroscope-go/upstream"
	"github.com/grafana/pyroscope-go/upstream/remote"
	"github.com/rs/zerolog"
)

var _ remote.Logger = (*zerologWrapper)(nil)

type zerologWrapper struct {
	logger zerolog.Logger
}

func (z zerologWrapper) Infof(f string, args ...interface{})  { z.logger.Info().Msgf(f, args...) }
func (z zerologWrapper) Debugf(f string, args ...interface{}) { z.logger.Debug().Msgf(f, args...) }
func (z zerologWrapper) Errorf(f string, args ...interface{}) { z.logger.Error().Msgf(f, args...) }

type PyroscopePusher struct {
	Address string
	Remote  *remote.Remote
	Logger  zerolog.Logger
}

func NewPusher(address string, logger zerolog.Logger) (*PyroscopePusher, error) {
	rmt, err := remote.NewRemote(remote.Config{
		AuthToken:         "",
		BasicAuthUser:     "",
		BasicAuthPassword: "",
		TenantID:          "",
		HTTPHeaders:       nil,
		Threads:           1,
		Address:           "http://192.168.86.122:4040/",
		Timeout:           time.Second * 20,
		Logger:            &zerologWrapper{logger: logger},
	})
	if err != nil {
		return nil, fmt.Errorf("new remote: %w", err)
	}

	go rmt.Start()
	return &PyroscopePusher{
		Address: address,
		Remote:  rmt,
		Logger:  logger,
	}, nil
}

func (p *PyroscopePusher) Stop() {
	p.Remote.Stop()
}

func (p *PyroscopePusher) Push(name string, pb *profile.Profile) error {
	var buf bytes.Buffer
	err := pb.Write(&buf)
	if err != nil {
		return fmt.Errorf("write proto: %w", err)
	}

	start := time.UnixMilli(pb.TimeNanos / 1e6)
	end := start.Add(time.Duration(pb.DurationNanos))

	p.Remote.Upload(&upstream.UploadJob{
		Name: name,
		// Fix this
		StartTime:       start,
		EndTime:         end,
		SpyName:         "",
		SampleRate:      0,
		Units:           "cpu",
		AggregationType: "sum",
		Format:          upstream.FormatPprof,
		Profile:         buf.Bytes(),
		// @emyrk: Unsure why we would need this?
		SampleTypeConfig: map[string]*upstream.SampleType{
			"cpu": {
				Units:       "nanoseconds",
				Aggregation: "sum",
				DisplayName: "cpu",
				// Not sampled, all the calls are present.
				Sampled:    false,
				Cumulative: false,
			},
			"samples": {
				Units:       "count",
				Aggregation: "sum",
				DisplayName: "Count",
				// Not sampled, all the calls are present.
				Sampled:    false,
				Cumulative: false,
			},
		},
	})

	return nil
}
