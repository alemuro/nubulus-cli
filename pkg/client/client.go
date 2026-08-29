package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Client is the main entry point to talk to Nubulus APIs.
type Client struct {
	DNS    *DNSClient
	Tunnel *TunnelClient
}

type service struct {
	base      string
	http      *http.Client
	userAgent string
}

// New creates an authenticated Nubulus Client.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.TokenURL == "" {
		cfg.TokenURL = DefaultTokenURL
	}
	if cfg.ProjectID == "" {
		cfg.ProjectID = DefaultProjectID
	}
	if cfg.DNSEndpoint == "" {
		cfg.DNSEndpoint = DefaultDNSEndpoint
	}
	if cfg.TunnelEndpoint == "" {
		cfg.TunnelEndpoint = DefaultTunnelEndpoint
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "nubulus-cli/1.0.0"
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return nil, fmt.Errorf("client_id i client_secret són obligatoris (especifiqueu-los per flags, fitxer de configuració o variables NUBULUS_CLIENT_ID / NUBULUS_CLIENT_SECRET)")
		}

		oauthCfg := &clientcredentials.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     cfg.TokenURL,
			Scopes:       Scopes(cfg.ProjectID),
			AuthStyle:    oauth2.AuthStyleInParams,
		}

		tokenCtx := context.WithValue(context.WithoutCancel(ctx), oauth2.HTTPClient, &http.Client{
			Timeout: 30 * time.Second,
		})
		httpClient = oauthCfg.Client(tokenCtx)
		httpClient.Timeout = 60 * time.Second
	}

	return &Client{
		DNS: &DNSClient{service: service{
			base:      strings.TrimSuffix(cfg.DNSEndpoint, "/"),
			http:      httpClient,
			userAgent: cfg.UserAgent,
		}},
		Tunnel: &TunnelClient{service: service{
			base:      strings.TrimSuffix(cfg.TunnelEndpoint, "/"),
			http:      httpClient,
			userAgent: cfg.UserAgent,
		}},
	}, nil
}

func (s *service) do(ctx context.Context, method, path string, in, out any) error {
	var bodyBytes []byte
	if in != nil {
		encoded, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("error codificant json: %w", err)
		}
		bodyBytes = encoded
	}

	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}

		var body io.Reader
		if bodyBytes != nil {
			body = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, s.base+path, body)
		if err != nil {
			return fmt.Errorf("error construint petició: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		if in != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if s.userAgent != "" {
			req.Header.Set("User-Agent", s.userAgent)
		}

		resp, err := s.http.Do(req)
		if err != nil {
			lastErr = &TransportError{Method: method, URL: s.base + path, Err: err}
			continue
		}

		if resp.StatusCode >= 400 {
			apiErr := parseAPIError(method, s.base+path, resp)
			resp.Body.Close()

			// Retry on transient 502/503/504 or ACCOUNTS_UNREACHABLE
			if resp.StatusCode == http.StatusServiceUnavailable ||
				resp.StatusCode == http.StatusBadGateway ||
				resp.StatusCode == http.StatusGatewayTimeout ||
				CodeOf(apiErr) == "ACCOUNTS_UNREACHABLE" {
				lastErr = apiErr
				continue
			}

			return apiErr
		}

		if out == nil || resp.StatusCode == http.StatusNoContent {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(out)
		resp.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("error descodificant resposta de %s %s: %w", method, path, decodeErr)
		}
		return nil
	}

	return lastErr
}
