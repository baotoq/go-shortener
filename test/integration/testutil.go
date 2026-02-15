//go:build integration

package integration

import (
  "context"
  "fmt"
  "io"
  "net/http"
  "os"
  "testing"
  "time"
)

// skipIfShort skips the test if running in short mode or if SKIP_INTEGRATION is set
func skipIfShort(t *testing.T) {
  t.Helper()
  if testing.Short() {
    t.Skip("Skipping integration test in short mode")
  }
  if os.Getenv("SKIP_INTEGRATION") != "" {
    t.Skip("Skipping integration test (SKIP_INTEGRATION set)")
  }
}

// waitForHealthy polls a health endpoint until it returns 200 or timeout is reached
func waitForHealthy(ctx context.Context, url string, timeout time.Duration) error {
  deadline := time.Now().Add(timeout)
  client := &http.Client{
    Timeout: 5 * time.Second,
  }

  for time.Now().Before(deadline) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
      return fmt.Errorf("create request: %w", err)
    }

    resp, err := client.Do(req)
    if err == nil {
      io.Copy(io.Discard, resp.Body)
      resp.Body.Close()
      if resp.StatusCode == http.StatusOK {
        return nil
      }
    }

    select {
    case <-ctx.Done():
      return ctx.Err()
    case <-time.After(2 * time.Second):
      // Continue polling
    }
  }

  return fmt.Errorf("timeout waiting for %s to be healthy", url)
}
