//go:build integration

package integration

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "os"
  "os/exec"
  "path/filepath"
  "testing"
  "time"

  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/require"
)

// TestDaprPubSub_ClickEventDeliveredToAnalytics verifies that click events
// published by URL Service are successfully received and processed by Analytics Service
// through real Dapr pub/sub (in-memory).
func TestDaprPubSub_ClickEventDeliveredToAnalytics(t *testing.T) {
  skipIfShort(t)

  // Start the Docker Compose stack
  setupDockerCompose(t)

  // Wait for services to be healthy
  baseURL := "http://localhost:8080"
  analyticsURL := "http://localhost:8081"

  err := waitForHealthy(t.Context(), baseURL+"/healthz", 90*time.Second)
  require.NoError(t, err, "URL Service failed to become healthy")

  err = waitForHealthy(t.Context(), analyticsURL+"/healthz", 90*time.Second)
  require.NoError(t, err, "Analytics Service failed to become healthy")

  // Additional wait for Dapr sidecars to be fully ready
  // The Dapr sidecars have a 30s start_period, so give them time
  time.Sleep(5 * time.Second)

  // Step 1: Create a short URL
  createReq := map[string]interface{}{
    "original_url": "https://example.com/test-dapr-integration",
  }
  createBody, err := json.Marshal(createReq)
  require.NoError(t, err)

  resp, err := http.Post(baseURL+"/api/v1/urls", "application/json", bytes.NewReader(createBody))
  require.NoError(t, err)
  defer resp.Body.Close()

  require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create short URL")

  var createResp struct {
    ShortCode   string `json:"short_code"`
    OriginalURL string `json:"original_url"`
    ShortURL    string `json:"short_url"`
  }
  err = json.NewDecoder(resp.Body).Decode(&createResp)
  require.NoError(t, err)
  require.NotEmpty(t, createResp.ShortCode)

  shortCode := createResp.ShortCode
  t.Logf("Created short URL with code: %s", shortCode)

  // Step 2: Visit the short URL (trigger redirect and click event)
  client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
      return http.ErrUseLastResponse // Don't follow redirects
    },
    Timeout: 10 * time.Second,
  }

  redirectResp, err := client.Get(baseURL + "/" + shortCode)
  require.NoError(t, err)
  io.Copy(io.Discard, redirectResp.Body)
  redirectResp.Body.Close()

  require.Equal(t, http.StatusFound, redirectResp.StatusCode, "Expected 302 redirect")
  location := redirectResp.Header.Get("Location")
  require.Equal(t, "https://example.com/test-dapr-integration", location)

  t.Logf("Visited short URL, triggered click event")

  // Step 3: Wait for event propagation through Dapr pub/sub
  // The event flow: URL Service publishes -> Dapr in-memory pub/sub -> Analytics Service receives
  time.Sleep(5 * time.Second)

  // Step 4: Query analytics to verify the click was recorded
  // Use polling to handle timing variations
  var totalClicks int
  success := false
  for i := 0; i < 10; i++ {
    analyticsResp, err := http.Get(analyticsURL + "/analytics/" + shortCode)
    if err == nil {
      defer analyticsResp.Body.Close()

      if analyticsResp.StatusCode == http.StatusOK {
        var analyticsData struct {
          ShortCode   string `json:"short_code"`
          TotalClicks int    `json:"total_clicks"`
        }
        if err := json.NewDecoder(analyticsResp.Body).Decode(&analyticsData); err == nil {
          totalClicks = analyticsData.TotalClicks
          if totalClicks >= 1 {
            success = true
            break
          }
        }
      }
    }

    if i < 9 {
      time.Sleep(2 * time.Second)
    }
  }

  require.True(t, success, "Click event was not delivered to Analytics Service within timeout")
  assert.GreaterOrEqual(t, totalClicks, 1, "Expected at least 1 click recorded")

  t.Logf("Successfully verified Dapr pub/sub: %d clicks recorded", totalClicks)
}

// TestDaprServiceInvocation_ClickCountEnrichment verifies that the URL Service
// successfully retrieves click counts from Analytics Service via Dapr service invocation.
func TestDaprServiceInvocation_ClickCountEnrichment(t *testing.T) {
  skipIfShort(t)

  // Start the Docker Compose stack
  setupDockerCompose(t)

  // Wait for services to be healthy
  baseURL := "http://localhost:8080"
  analyticsURL := "http://localhost:8081"

  err := waitForHealthy(t.Context(), baseURL+"/healthz", 90*time.Second)
  require.NoError(t, err, "URL Service failed to become healthy")

  err = waitForHealthy(t.Context(), analyticsURL+"/healthz", 90*time.Second)
  require.NoError(t, err, "Analytics Service failed to become healthy")

  // Additional wait for Dapr sidecars
  time.Sleep(5 * time.Second)

  // Step 1: Create a short URL
  createReq := map[string]interface{}{
    "original_url": "https://example.com/test-service-invocation",
  }
  createBody, err := json.Marshal(createReq)
  require.NoError(t, err)

  resp, err := http.Post(baseURL+"/api/v1/urls", "application/json", bytes.NewReader(createBody))
  require.NoError(t, err)
  defer resp.Body.Close()

  require.Equal(t, http.StatusCreated, resp.StatusCode)

  var createResp struct {
    ShortCode string `json:"short_code"`
  }
  err = json.NewDecoder(resp.Body).Decode(&createResp)
  require.NoError(t, err)

  shortCode := createResp.ShortCode
  t.Logf("Created short URL with code: %s", shortCode)

  // Step 2: Visit the short URL to generate a click
  client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
      return http.ErrUseLastResponse
    },
    Timeout: 10 * time.Second,
  }

  redirectResp, err := client.Get(baseURL + "/" + shortCode)
  require.NoError(t, err)
  io.Copy(io.Discard, redirectResp.Body)
  redirectResp.Body.Close()

  require.Equal(t, http.StatusFound, redirectResp.StatusCode)

  // Step 3: Wait for event processing
  time.Sleep(5 * time.Second)

  // Step 4: Query link detail endpoint (uses Dapr service invocation)
  // This endpoint calls Analytics Service via Dapr to enrich with click count
  var totalClicks int
  success := false
  for i := 0; i < 10; i++ {
    linkResp, err := http.Get(baseURL + "/api/v1/links/" + shortCode)
    if err == nil {
      defer linkResp.Body.Close()

      if linkResp.StatusCode == http.StatusOK {
        var linkData struct {
          ShortCode   string `json:"short_code"`
          OriginalURL string `json:"original_url"`
          TotalClicks int    `json:"total_clicks"`
        }
        if err := json.NewDecoder(linkResp.Body).Decode(&linkData); err == nil {
          totalClicks = linkData.TotalClicks
          if totalClicks >= 1 {
            success = true
            break
          }
        }
      }
    }

    if i < 9 {
      time.Sleep(2 * time.Second)
    }
  }

  require.True(t, success, "Service invocation did not return click count within timeout")
  assert.GreaterOrEqual(t, totalClicks, 1, "Expected at least 1 click via service invocation")

  t.Logf("Successfully verified Dapr service invocation: %d clicks retrieved", totalClicks)
}

// setupDockerCompose starts the Docker Compose stack for integration testing
func setupDockerCompose(t *testing.T) {
  t.Helper()

  // Find project root (contains docker-compose.yml)
  projectRoot, err := findProjectRoot()
  require.NoError(t, err, "Failed to find project root")

  // Clean up any existing containers from previous failed runs
  cleanupCmd := exec.Command("docker", "compose", "-f", filepath.Join(projectRoot, "docker-compose.yml"), "down", "-v", "--remove-orphans")
  cleanupCmd.Dir = projectRoot
  _ = cleanupCmd.Run() // Ignore errors from cleanup

  // Start the stack
  t.Log("Starting Docker Compose stack...")
  upCmd := exec.Command("docker", "compose", "-f", filepath.Join(projectRoot, "docker-compose.yml"), "up", "--build", "-d")
  upCmd.Dir = projectRoot
  upCmd.Stdout = os.Stdout
  upCmd.Stderr = os.Stderr

  err = upCmd.Run()
  require.NoError(t, err, "Failed to start Docker Compose stack")

  t.Log("Docker Compose stack started")

  // Register cleanup
  t.Cleanup(func() {
    t.Log("Stopping Docker Compose stack...")
    downCmd := exec.Command("docker", "compose", "-f", filepath.Join(projectRoot, "docker-compose.yml"), "down", "-v", "--remove-orphans")
    downCmd.Dir = projectRoot
    downCmd.Stdout = os.Stdout
    downCmd.Stderr = os.Stderr
    _ = downCmd.Run()
    t.Log("Docker Compose stack stopped")
  })
}

// findProjectRoot locates the project root directory (contains docker-compose.yml)
func findProjectRoot() (string, error) {
  // Start from current working directory
  dir, err := os.Getwd()
  if err != nil {
    return "", err
  }

  // Walk up the directory tree
  for {
    composeFile := filepath.Join(dir, "docker-compose.yml")
    if _, err := os.Stat(composeFile); err == nil {
      return dir, nil
    }

    parent := filepath.Dir(dir)
    if parent == dir {
      // Reached root without finding docker-compose.yml
      return "", fmt.Errorf("docker-compose.yml not found in directory tree")
    }
    dir = parent
  }
}
