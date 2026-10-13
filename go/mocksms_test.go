package mocksms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// waitForOTP against a stub server needs no Docker daemon.
func TestWaitForOTPStub(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"482916","source":"extracted"}`))
	}))
	defer server.Close()

	c := &Container{BaseURL: server.URL}
	code, err := c.WaitForOTP(context.Background(), "+15005550006", 5*time.Second)
	if err != nil {
		t.Fatalf("WaitForOTP: %v", err)
	}
	if code != "482916" {
		t.Fatalf("expected 482916, got %q", code)
	}
}

func TestWaitForOTPTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found"}}`))
	}))
	defer server.Close()

	c := &Container{BaseURL: server.URL}
	if _, err := c.WaitForOTP(context.Background(), "+19999999999", 1200*time.Millisecond); err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

// TestContainerRoundTrip needs a Docker daemon (CI); skipped elsewhere.
func TestContainerRoundTrip(t *testing.T) {
	if os.Getenv("MOCKSMS_TEST_DOCKER") == "" {
		t.Skip("needs Docker daemon")
	}
	ctx := context.Background()
	c, err := Start(ctx, Options{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = c.Stop(ctx) }()

	if err := c.SendSMS(ctx, "+15555550100", "+15005550006", "Your verification code is 731946"); err != nil {
		t.Fatalf("SendSMS: %v", err)
	}
	code, err := c.WaitForOTP(ctx, "+15005550006", 30*time.Second)
	if err != nil {
		t.Fatalf("WaitForOTP: %v", err)
	}
	if code != "731946" {
		t.Fatalf("expected 731946, got %q", code)
	}
}
