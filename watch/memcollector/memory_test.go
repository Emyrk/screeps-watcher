package memcollector_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Emyrk/screeps-watcher/watch/memcollector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollector(t *testing.T) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})
	c := memcollector.New(logger, "test", prometheus.Labels{"test": "test"})
	data, err := os.ReadFile("testdata/full/memory.json")
	require.NoError(t, err)

	count, err := c.SetMemory(data)
	require.NoError(t, err)
	require.Greater(t, count, 100)

	reg := prometheus.NewRegistry()
	reg.MustRegister(c)
	fmt.Println(RegistryDump(reg))
}

// updateGoldenFiles is a flag that can be set to update golden files.
var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

func TestGoldenFiles(t *testing.T) {
	dirs, err := os.ReadDir("testdata")
	require.NoError(t, err)
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		t.Run(dir.Name(), func(t *testing.T) {
			require.True(t, dir.IsDir())
			memory := filepath.Join("testdata", dir.Name(), "memory.json")
			prom := filepath.Join("testdata", dir.Name(), "prometheus.txt")
			memoryJSON, err := os.ReadFile(memory)
			require.NoError(t, err)

			promData, err := os.ReadFile(prom)
			require.NoError(t, err)

			c := memcollector.New(logger, "test", prometheus.Labels{"test": "test"})
			c.SetNow(func() time.Time {
				return time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
			})
			_, err = c.SetMemory(memoryJSON)
			require.NoError(t, err)

			reg := prometheus.NewRegistry()
			reg.MustRegister(c)

			found := RegistryDump(reg)
			if *updateGoldenFiles {
				err := os.WriteFile(prom, []byte(found), 0644)
				require.NoError(t, err)

				var pretty bytes.Buffer
				err = json.Indent(&pretty, memoryJSON, "", "  ")
				if err == nil {
					_ = os.WriteFile(memory, pretty.Bytes(), 0644)
				}
			} else {
				if !assert.Equal(t, string(promData), found) {
					t.Logf("Found:\n%s\n", found)
				}
			}
		})
	}

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
