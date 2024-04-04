package memcollector_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Emyrk/screeps-watcher/watch/memcollector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestCollector(t *testing.T) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})
	c := memcollector.New(logger, "test", prometheus.Labels{"test": "test"})
	data, err := os.ReadFile("testdata/memory.example.json")
	require.NoError(t, err)

	count, err := c.SetMemory(data)
	require.NoError(t, err)
	require.Greater(t, count, 100)

	reg := prometheus.NewRegistry()
	reg.MustRegister(c)
	fmt.Println(RegistryDump(reg))
}

func RegistryDump(reg prometheus.Gatherer) string {
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	rec := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)
	resp := rec.Result()
	data, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return string(data)
}
