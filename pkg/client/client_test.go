package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := New(context.Background(), Config{
		DNSEndpoint:    server.URL,
		TunnelEndpoint: server.URL,
		HTTPClient:     server.Client(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encoding error: %v", err)
	}
}

func TestDNSZonesAndRRsets(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/zones":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"data": []map[string]any{
					{"id": "zone_1", "name": "example.com", "status": "active"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/zones":
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"zone": map[string]any{
					"id": "zone_2", "name": "new.com", "status": "pending_verification",
				},
				"verification": map[string]any{
					"zone": "new.com", "required": true, "txt_record_host": "_nubulus-challenge.new.com",
					"txt_record_value": "tok123",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/zones/example.com/rrsets":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"zone": "example.com", "serial": 100,
				"rrsets": []map[string]any{
					{"name": "www.example.com.", "type": "A", "ttl": 300, "values": []string{"1.2.3.4"}},
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/zones/example.com/rrsets/www.example.com./A":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"name": "www.example.com.", "type": "A", "ttl": 600, "values": []string{"1.2.3.4", "5.6.7.8"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/zones/example.com/verify":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"zone": "example.com", "status": "active", "verified": true, "method": "txt",
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))

	ctx := context.Background()

	// List zones
	zones, err := client.DNS.ListZones(ctx)
	if err != nil || len(zones) != 1 || zones[0].Name != "example.com" {
		t.Fatalf("ListZones failed: %v, %v", err, zones)
	}

	// Create zone
	created, err := client.DNS.CreateZone(ctx, "new.com")
	if err != nil || created.Zone.ID != "zone_2" || created.Verification == nil {
		t.Fatalf("CreateZone failed: %v, %v", err, created)
	}

	// Verify zone
	verified, err := client.DNS.Verify(ctx, "example.com")
	if err != nil || !verified.Verified {
		t.Fatalf("Verify failed: %v, %v", err, verified)
	}

	// Get RRset
	rrset, err := client.DNS.GetRRset(ctx, "example.com", "www.example.com.", "A")
	if err != nil || rrset.Values[0] != "1.2.3.4" {
		t.Fatalf("GetRRset failed: %v, %v", err, rrset)
	}

	// Put RRset
	updated, err := client.DNS.PutRRset(ctx, "example.com", "www.example.com.", "A", 600, []string{"1.2.3.4", "5.6.7.8"})
	if err != nil || updated.TTL != 600 || len(updated.Values) != 2 {
		t.Fatalf("PutRRset failed: %v, %v", err, updated)
	}
}

func TestTunnelsAndRoutes(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/tunnels":
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"tunnel_id":        "tun-1",
				"tunnel_token":     "tok-secret",
				"tunnel_subdomain": "tun-1.example.net",
				"cname_target":     "tun-1.example.net.",
				"wireguard_ip":     "10.0.0.5",
				"wireguard": map[string]any{
					"interface": map[string]any{
						"private_key": "priv-key",
						"address":     "10.0.0.5/32",
					},
					"peer": map[string]any{
						"public_key": "pub-key",
						"endpoint":   "gw.example.net:51820",
					},
				},
				"adopted": false,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/tunnels/tun-1":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"tunnel": map[string]any{
					"id": "tun-1", "status": "active", "online_status": "online",
				},
				"routes": []map[string]any{
					{
						"id": "rt-1", "tunnel_id": "tun-1", "hostname": "app.example.com",
						"upstream_host": "127.0.0.1", "upstream_port": 8080,
					},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/tunnels/tun-1/routes":
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"id": "rt-2", "tunnel_id": "tun-1", "hostname": "api.example.com",
				"upstream_host": "10.0.0.2", "upstream_port": 3000,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/tunnels/tun-1/rotate-token":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"tunnel_id": "tun-1", "tunnel_token": "tok-rotated",
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))

	ctx := context.Background()

	// Create tunnel
	tun, err := client.Tunnel.CreateTunnel(ctx, CreateTunnelInput{Name: "prod"})
	if err != nil || tun.TunnelID != "tun-1" || tun.TunnelToken != "tok-secret" {
		t.Fatalf("CreateTunnel failed: %v, %v", err, tun)
	}

	// WireGuard config formatting
	confStr := FormatWireGuardConfig(tun.WireGuard)
	if len(confStr) == 0 {
		t.Fatal("FormatWireGuardConfig returned empty string")
	}

	// Get tunnel
	detail, err := client.Tunnel.GetTunnel(ctx, "tun-1")
	if err != nil || detail.Tunnel.ID != "tun-1" || len(detail.Routes) != 1 {
		t.Fatalf("GetTunnel failed: %v, %v", err, detail)
	}

	// Create route
	rt, err := client.Tunnel.CreateRoute(ctx, "tun-1", CreateRouteInput{
		Type:         "host",
		Hostname:     "api.example.com",
		UpstreamHost: "10.0.0.2",
		UpstreamPort: 3000,
	})
	if err != nil || rt.ID != "rt-2" {
		t.Fatalf("CreateRoute failed: %v, %v", err, rt)
	}

	// Rotate token
	rot, err := client.Tunnel.RotateToken(ctx, "tun-1")
	if err != nil || rot.TunnelToken != "tok-rotated" {
		t.Fatalf("RotateToken failed: %v, %v", err, rot)
	}
}
