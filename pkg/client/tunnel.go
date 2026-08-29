package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type TunnelClient struct {
	service
}

type Tunnel struct {
	ID                 string     `json:"id" yaml:"id"`
	AccountID          string     `json:"account_id" yaml:"account_id"`
	UserID             string     `json:"user_id" yaml:"user_id"`
	Name               string     `json:"name,omitempty" yaml:"name,omitempty"`
	ExternalID         string     `json:"external_id,omitempty" yaml:"external_id,omitempty"`
	TunnelSubdomain    string     `json:"tunnel_subdomain" yaml:"tunnel_subdomain"`
	WireGuardIP        string     `json:"wireguard_ip" yaml:"wireguard_ip"`
	WireGuardPublicKey string     `json:"wireguard_public_key,omitempty" yaml:"wireguard_public_key,omitempty"`
	Status             string     `json:"status" yaml:"status"`
	OnlineStatus       string     `json:"online_status" yaml:"online_status"`
	LastHandshakeAt    *time.Time `json:"last_handshake_at,omitempty" yaml:"last_handshake_at,omitempty"`
	StatusChangedAt    *time.Time `json:"status_changed_at,omitempty" yaml:"status_changed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" yaml:"updated_at"`
}

type TunnelSummary struct {
	Tunnel
	RouteCount int `json:"route_count" yaml:"route_count"`
}

type TunnelWithRoutes struct {
	Tunnel *Tunnel `json:"tunnel" yaml:"tunnel"`
	Routes []Route `json:"routes" yaml:"routes"`
}

type CreateTunnelResult struct {
	TunnelID        string          `json:"tunnel_id" yaml:"tunnel_id"`
	TunnelToken     string          `json:"tunnel_token,omitempty" yaml:"tunnel_token,omitempty"`
	TunnelSubdomain string          `json:"tunnel_subdomain" yaml:"tunnel_subdomain"`
	CNAMETarget     string          `json:"cname_target" yaml:"cname_target"`
	Instructions    string          `json:"instructions,omitempty" yaml:"instructions,omitempty"`
	WireGuard       WireGuardConfig `json:"wireguard,omitempty" yaml:"wireguard,omitempty"`
	WireGuardIP     string          `json:"wireguard_ip" yaml:"wireguard_ip"`
	Adopted         bool            `json:"adopted" yaml:"adopted"`
}

type CreateTunnelInput struct {
	Name       string `json:"name,omitempty"`
	ExternalID string `json:"external_id,omitempty"`
}

type RotateTokenResult struct {
	TunnelID    string `json:"tunnel_id" yaml:"tunnel_id"`
	TunnelToken string `json:"tunnel_token" yaml:"tunnel_token"`
}

type WireGuardConfig struct {
	Interface WireGuardInterface `json:"interface" yaml:"interface"`
	Peer      WireGuardPeer      `json:"peer" yaml:"peer"`
}

type WireGuardInterface struct {
	PrivateKey string `json:"private_key,omitempty" yaml:"private_key,omitempty"`
	Address    string `json:"address" yaml:"address"`
	DNS        string `json:"dns,omitempty" yaml:"dns,omitempty"`
}

type WireGuardPeer struct {
	PublicKey           string `json:"public_key" yaml:"public_key"`
	Endpoint            string `json:"endpoint" yaml:"endpoint"`
	AllowedIPs          string `json:"allowed_ips" yaml:"allowed_ips"`
	PersistentKeepalive int    `json:"persistent_keepalive,omitempty" yaml:"persistent_keepalive,omitempty"`
}

type Route struct {
	ID             string    `json:"id" yaml:"id"`
	TunnelID       string    `json:"tunnel_id" yaml:"tunnel_id"`
	Type           string    `json:"type" yaml:"type"`
	Hostname       string    `json:"hostname" yaml:"hostname"`
	PathPrefix     string    `json:"path_prefix" yaml:"path_prefix"`
	UpstreamHost   string    `json:"upstream_host" yaml:"upstream_host"`
	UpstreamPort   int       `json:"upstream_port" yaml:"upstream_port"`
	UpstreamScheme string    `json:"upstream_scheme" yaml:"upstream_scheme"`
	StripPrefix    bool      `json:"strip_prefix" yaml:"strip_prefix"`
	Enabled        bool      `json:"enabled" yaml:"enabled"`
	Priority       int       `json:"priority" yaml:"priority"`
	CreatedAt      time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" yaml:"updated_at"`
}

type CreateRouteInput struct {
	Type           string `json:"type"`
	Hostname       string `json:"hostname"`
	PathPrefix     string `json:"path_prefix,omitempty"`
	UpstreamHost   string `json:"upstream_host"`
	UpstreamPort   int    `json:"upstream_port"`
	UpstreamScheme string `json:"upstream_scheme,omitempty"`
	StripPrefix    bool   `json:"strip_prefix"`
	Priority       int    `json:"priority,omitempty"`
}

type UpdateRouteInput struct {
	UpstreamHost   *string `json:"upstream_host,omitempty"`
	UpstreamPort   *int    `json:"upstream_port,omitempty"`
	UpstreamScheme *string `json:"upstream_scheme,omitempty"`
	StripPrefix    *bool   `json:"strip_prefix,omitempty"`
	Priority       *int    `json:"priority,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

const tunnelPageSize = 100
const listPageLimit = 1000

// CreateTunnel creates a new tunnel or adopts an existing one by external_id.
func (c *TunnelClient) CreateTunnel(ctx context.Context, in CreateTunnelInput) (*CreateTunnelResult, error) {
	var out CreateTunnelResult
	if err := c.do(ctx, http.MethodPost, "/api/v2/tunnels", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RotateToken generates a new secret token for a tunnel.
func (c *TunnelClient) RotateToken(ctx context.Context, id string) (*RotateTokenResult, error) {
	var out RotateTokenResult
	path := "/api/v2/tunnels/" + url.PathEscape(id) + "/rotate-token"
	if err := c.do(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// FindTunnelByExternalID finds a single tunnel by creator's external_id.
func (c *TunnelClient) FindTunnelByExternalID(ctx context.Context, externalID string) (*TunnelSummary, error) {
	var out struct {
		Data []TunnelSummary `json:"data"`
	}
	path := "/api/v2/tunnels?external_id=" + url.QueryEscape(externalID)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, nil
	}
	return &out.Data[0], nil
}

// GetTunnel returns a tunnel and all its routes.
func (c *TunnelClient) GetTunnel(ctx context.Context, id string) (*TunnelWithRoutes, error) {
	var out TunnelWithRoutes
	if err := c.do(ctx, http.MethodGet, "/api/v2/tunnels/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTunnels returns all tunnels for the account with auto-pagination.
func (c *TunnelClient) ListTunnels(ctx context.Context) ([]TunnelSummary, error) {
	var all []TunnelSummary
	offset := 0

	for page := 0; page < listPageLimit; page++ {
		var out struct {
			Data   []TunnelSummary `json:"data"`
			Limit  int             `json:"limit"`
			Offset int             `json:"offset"`
		}
		path := "/api/v2/tunnels?limit=" + strconv.Itoa(tunnelPageSize) +
			"&offset=" + strconv.Itoa(offset)
		if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
			return nil, err
		}
		if len(out.Data) == 0 {
			return all, nil
		}
		all = append(all, out.Data...)
		offset += len(out.Data)
	}

	return all, nil
}

// DeleteTunnel deletes a tunnel and all routes attached to it.
func (c *TunnelClient) DeleteTunnel(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v2/tunnels/"+url.PathEscape(id), nil, nil)
}

// CreateRoute adds a route to a tunnel.
func (c *TunnelClient) CreateRoute(ctx context.Context, tunnelID string, in CreateRouteInput) (*Route, error) {
	var out Route
	if err := c.do(ctx, http.MethodPost, routesPath(tunnelID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetRoute gets a single route.
func (c *TunnelClient) GetRoute(ctx context.Context, tunnelID, routeID string) (*Route, error) {
	var out Route
	if err := c.do(ctx, http.MethodGet, routePath(tunnelID, routeID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRoutes lists all routes for a tunnel.
func (c *TunnelClient) ListRoutes(ctx context.Context, tunnelID string) ([]Route, error) {
	var out struct {
		Routes []Route `json:"routes"`
		Total  int     `json:"total"`
	}
	if err := c.do(ctx, http.MethodGet, routesPath(tunnelID), nil, &out); err != nil {
		return nil, err
	}
	return out.Routes, nil
}

// UpdateRoute modifies editable fields of a route.
func (c *TunnelClient) UpdateRoute(ctx context.Context, tunnelID, routeID string, in UpdateRouteInput) (*Route, error) {
	var out Route
	if err := c.do(ctx, http.MethodPut, routePath(tunnelID, routeID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteRoute deletes a route.
func (c *TunnelClient) DeleteRoute(ctx context.Context, tunnelID, routeID string) error {
	return c.do(ctx, http.MethodDelete, routePath(tunnelID, routeID), nil, nil)
}

func routesPath(tunnelID string) string {
	return "/api/v2/tunnels/" + url.PathEscape(tunnelID) + "/routes"
}

func routePath(tunnelID, routeID string) string {
	return routesPath(tunnelID) + "/" + url.PathEscape(routeID)
}
