package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Emyrk/screeps-watcher/watch/memory"
)

// https://github.com/screepers/node-screeps-api/blob/master/docs/Endpoints.md
func (w *Watcher) MemorySegment(ctx context.Context, id int, shard string) (json.RawMessage, error) {
	vals := url.Values{
		"segment": []string{strconv.Itoa(id)},
		"shard":   []string{shard},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", w.URL.ResolveReference(&url.URL{
		Path:     "/api/user/memory-segment",
		RawQuery: vals.Encode(),
	}).String(), nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := w.AuthMethod.AuthenticatedRequest(w.cli, req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		sleepUntil := time.Now().Add(time.Minute * 10)
		reset := resp.Header.Get("X-RateLimit-Reset")
		if reset != "" {
			resetAt, err := strconv.ParseInt(reset, 10, 64)
			if err == nil {
				sleepUntil = time.Unix(resetAt, 0).Add(time.Second * 5)
			}
			w.logger.Error().
				Time("reset", sleepUntil).
				Msg("rate limit hit")
		}
		w.rateLimitUntil = sleepUntil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}

	return memory.Decode(respData)
}
