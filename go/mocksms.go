// Package mocksms starts the mocksms sandbox in a Testcontainer for Go
// integration tests, with a WaitForOTP helper.
package mocksms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DefaultImage is used when Start gets no explicit image.
const DefaultImage = "ghcr.io/aeomar999/commpit/mocksms:latest"

// Options configures the container.
type Options struct {
	// Image to run. Defaults to DefaultImage.
	Image string
	// OTPCode pins every verification code. Defaults to "123456".
	OTPCode string
}

// Container is a running mocksms sandbox.
type Container struct {
	tc      *testcontainers.DockerContainer
	BaseURL string
}

// Start launches mocksms (in-memory store) and waits until healthy.
func Start(ctx context.Context, opts Options) (*Container, error) {
	image := opts.Image
	if image == "" {
		// MOCKSMS_IMAGE overrides the image (CI matrix, local builds).
		image = os.Getenv("MOCKSMS_IMAGE")
	}
	if image == "" {
		image = DefaultImage
	}
	otpCode := opts.OTPCode
	if otpCode == "" {
		otpCode = "123456"
	}
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"4010/tcp", "1025/tcp"},
		Env: map[string]string{
			"MOCKSMS_MEMORY":    "true",
			"MOCKSMS_OTP_CODE":  otpCode,
			"MOCKSMS_HTTP_HOST": "0.0.0.0",
			"MOCKSMS_SMTP_HOST": "0.0.0.0",
		},
		WaitingFor: wait.ForHTTP("/api/v1/healthz").WithPort("4010/tcp"),
	}
	tc, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("mocksms: start container: %w", err)
	}
	docker, ok := tc.(*testcontainers.DockerContainer)
	if !ok {
		return nil, fmt.Errorf("mocksms: unexpected container type %T", tc)
	}
	host, err := docker.Host(ctx)
	if err != nil {
		_ = docker.Terminate(ctx)
		return nil, fmt.Errorf("mocksms: host: %w", err)
	}
	port, err := docker.MappedPort(ctx, "4010")
	if err != nil {
		_ = docker.Terminate(ctx)
		return nil, fmt.Errorf("mocksms: port: %w", err)
	}
	return &Container{tc: docker, BaseURL: fmt.Sprintf("http://%s:%s", host, port.Port())}, nil
}

// Stop terminates the container.
func (c *Container) Stop(ctx context.Context) error {
	return c.tc.Terminate(ctx)
}

// SendSMS sends one SMS through the sandbox.
func (c *Container) SendSMS(ctx context.Context, from, to, body string) error {
	payload, err := json.Marshal(map[string]string{"from": from, "to": to, "body": body})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/sms", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mocksms", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("mocksms: send SMS: status %d: %s", resp.StatusCode, raw)
	}
	return nil
}

func getJSON(ctx context.Context, rawURL string, params map[string]string) (int, []byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return 0, nil, err
	}
	query := parsed.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-Mocksms", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	return resp.StatusCode, body, err
}

// WaitForOTP polls /otp/latest until a code appears or timeout elapses.
func (c *Container) WaitForOTP(ctx context.Context, to string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		status, body, err := getJSON(ctx, c.BaseURL+"/api/v1/otp/latest", map[string]string{"to": to})
		if err == nil && status == http.StatusOK {
			var resp struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(body, &resp); err == nil && resp.Code != "" {
				return resp.Code, nil
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return "", fmt.Errorf("mocksms: waitForOTP(%s): timed out after %s", to, timeout)
			}
		}
	}
}
