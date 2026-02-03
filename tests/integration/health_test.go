package integration

import (
	"net/http"
	"os"
	"testing"
)

func TestHealth(t *testing.T) {
	addr := os.Getenv("TEST_SERVER_ADDR")
	t.Log("addr:", addr)
	if addr == "" {
		addr = "http://localhost:8081"
	}

	resp, err := http.Get(addr + "/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
