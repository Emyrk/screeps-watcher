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

// https://screeps.com/api/game/market/stats?resourceType=energy&shard=shard3
func (w *Watcher) Market(ctx context.Context, resourceType string, shard string) (json.RawMessage, error) {
	vals := url.Values{
		"resourceType": []string{resourceType},
	}

	if shard != "" {
		vals.Set("shard", shard)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", w.URL.ResolveReference(&url.URL{
		Path:     "/api/game/market/stats",
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
		w.marketApiRateLimitUntil = w.rateLimtResetAt(resp)
		return nil, fmt.Errorf("rate limit hit")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}
	return respData, nil
}

func (w *Watcher) RoomObjects(ctx context.Context, room string, shard string) (json.RawMessage, error) {
	vals := url.Values{
		"room":  []string{room},
		"shard": []string{shard},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", w.URL.ResolveReference(&url.URL{
		Path:     "/api/game/room-objects",
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
		w.memorySegmentRateLimitUntil = w.rateLimtResetAt(resp)
		return nil, fmt.Errorf("rate limit hit")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}

	return respData, nil
}

func (w *Watcher) RoomOverview(ctx context.Context, room string, shard string) (json.RawMessage, error) {
	vals := url.Values{
		"interval": []string{"1"},
		"room":     []string{room},
		"shard":    []string{shard},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", w.URL.ResolveReference(&url.URL{
		Path:     "/api/game/room-overview",
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
		w.memorySegmentRateLimitUntil = w.rateLimtResetAt(resp)
		return nil, fmt.Errorf("rate limit hit")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}

	return respData, nil
}

func (w *Watcher) RoomTerrain(ctx context.Context, room string, shard string) (json.RawMessage, error) {
	//https://screeps.com/api/game/room-terrain?encoded=1&room=E11S53&shard=shard3
	vals := url.Values{
		"encoded": []string{"1"},
		"room":    []string{room},
		"shard":   []string{shard},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", w.URL.ResolveReference(&url.URL{
		Path:     "/api/game/room-terrain",
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
		w.memorySegmentRateLimitUntil = w.rateLimtResetAt(resp)
		return nil, fmt.Errorf("rate limit hit")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}

	return respData, nil
}

// https://github.com/screepers/node-screeps-api/blob/master/docs/Endpoints.md
func (w *Watcher) MemorySegment(ctx context.Context, id int, shard string) (json.RawMessage, int, error) {
	vals := url.Values{
		"segment": []string{strconv.Itoa(id)},
		"shard":   []string{shard},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", w.URL.ResolveReference(&url.URL{
		Path:     "/api/user/memory-segment",
		RawQuery: vals.Encode(),
	}).String(), nil)
	if err != nil {
		return nil, -1, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := w.AuthMethod.AuthenticatedRequest(w.cli, req)
	if err != nil {
		return nil, -1, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		w.memorySegmentRateLimitUntil = w.rateLimtResetAt(resp)
		return nil, -1, fmt.Errorf("rate limit hit")
	}

	if resp.StatusCode != 200 {
		return nil, -1, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, -1, fmt.Errorf("read all: %w", err)
	}

	decoded, err := memory.Decode(respData)
	if err != nil {
		return nil, -1, fmt.Errorf("decode: %w", err)
	}
	return decoded, len(respData), nil
}

func (w *Watcher) rateLimtResetAt(resp *http.Response) time.Time {
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
	return sleepUntil
}
