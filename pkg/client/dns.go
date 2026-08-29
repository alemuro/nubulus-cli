package client

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DNSClient struct {
	service
}

type Zone struct {
	ID            string     `json:"id" yaml:"id"`
	Name          string     `json:"name" yaml:"name"`
	Source        string     `json:"source" yaml:"source"`
	Status        string     `json:"status" yaml:"status"`
	AccountID     string     `json:"account_id" yaml:"account_id"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty" yaml:"verified_at,omitempty"`
	ReservedUntil *time.Time `json:"reserved_until,omitempty" yaml:"reserved_until,omitempty"`
	CreatedAt     time.Time  `json:"created_at" yaml:"created_at"`
	CreatedBy     string     `json:"created_by,omitempty" yaml:"created_by,omitempty"`
}

type ZoneDetail struct {
	Zone         *Zone             `json:"zone" yaml:"zone"`
	Serial       *uint32           `json:"serial,omitempty" yaml:"serial,omitempty"`
	Nameservers  []string          `json:"nameservers,omitempty" yaml:"nameservers,omitempty"`
	ReadAt       *time.Time        `json:"read_at,omitempty" yaml:"read_at,omitempty"`
	PrimaryError string            `json:"primary_error,omitempty" yaml:"primary_error,omitempty"`
	Verification *ZoneVerification `json:"verification,omitempty" yaml:"verification,omitempty"`
}

type ZoneVerification struct {
	Zone           string     `json:"zone" yaml:"zone"`
	Status         string     `json:"status" yaml:"status"`
	Source         string     `json:"source" yaml:"source"`
	Required       bool       `json:"required" yaml:"required"`
	TXTRecordHost  string     `json:"txt_record_host,omitempty" yaml:"txt_record_host,omitempty"`
	TXTRecordValue string     `json:"txt_record_value,omitempty" yaml:"txt_record_value,omitempty"`
	Nameservers    []string   `json:"nameservers" yaml:"nameservers"`
	ReservedUntil  *time.Time `json:"reserved_until,omitempty" yaml:"reserved_until,omitempty"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty" yaml:"verified_at,omitempty"`
}

type VerificationResult struct {
	Zone       string    `json:"zone" yaml:"zone"`
	Status     string    `json:"status" yaml:"status"`
	Verified   bool      `json:"verified" yaml:"verified"`
	Method     string    `json:"method,omitempty" yaml:"method,omitempty"`
	ReasonCode string    `json:"reason_code,omitempty" yaml:"reason_code,omitempty"`
	Reason     string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	CheckedAt  time.Time `json:"checked_at" yaml:"checked_at"`
}

type RRset struct {
	Name   string   `json:"name" yaml:"name"`
	Type   string   `json:"type" yaml:"type"`
	TTL    uint32   `json:"ttl" yaml:"ttl"`
	Values []string `json:"values" yaml:"values"`
}

type ZoneContent struct {
	Zone   string    `json:"zone" yaml:"zone"`
	Serial uint32    `json:"serial" yaml:"serial"`
	RRsets []RRset   `json:"rrsets" yaml:"rrsets"`
	ReadAt time.Time `json:"read_at" yaml:"read_at"`
}

func (c *ZoneContent) FindRRset(name, rrtype string) *RRset {
	if c == nil {
		return nil
	}
	name = NormalizeOwnerName(name)
	rrtype = strings.ToUpper(strings.TrimSpace(rrtype))
	for i := range c.RRsets {
		if NormalizeOwnerName(c.RRsets[i].Name) == name && c.RRsets[i].Type == rrtype {
			found := c.RRsets[i]
			found.Values = append([]string(nil), c.RRsets[i].Values...)
			return &found
		}
	}
	return nil
}

// CreateZone claims a domain name.
func (c *DNSClient) CreateZone(ctx context.Context, name string) (*ZoneDetail, error) {
	var out ZoneDetail
	body := map[string]string{"name": name}
	if err := c.do(ctx, http.MethodPost, "/api/v1/zones", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetZone gets one zone details.
func (c *DNSClient) GetZone(ctx context.Context, name string) (*ZoneDetail, error) {
	var out ZoneDetail
	if err := c.do(ctx, http.MethodGet, "/api/v1/zones/"+url.PathEscape(name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListZones lists all zones for the account.
func (c *DNSClient) ListZones(ctx context.Context) ([]Zone, error) {
	var out struct {
		Data []Zone `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/zones", nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// DeleteZone permanently deletes a zone.
func (c *DNSClient) DeleteZone(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/zones/"+url.PathEscape(name), nil, nil)
}

// Verification returns current verification challenge and instructions.
func (c *DNSClient) Verification(ctx context.Context, zone string) (*ZoneVerification, error) {
	var out ZoneVerification
	path := "/api/v1/zones/" + url.PathEscape(zone) + "/verification"
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Verify triggers an active verification check.
func (c *DNSClient) Verify(ctx context.Context, zone string) (*VerificationResult, error) {
	var out VerificationResult
	path := "/api/v1/zones/" + url.PathEscape(zone) + "/verify"
	if err := c.do(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ZoneContent reads the complete zone and its RRsets.
func (c *DNSClient) ZoneContent(ctx context.Context, zone string) (*ZoneContent, error) {
	var out ZoneContent
	path := "/api/v1/zones/" + url.PathEscape(zone) + "/rrsets"
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetRRset retrieves a single RRset by name and type.
func (c *DNSClient) GetRRset(ctx context.Context, zone, name, rrtype string) (*RRset, error) {
	content, err := c.ZoneContent(ctx, zone)
	if err != nil {
		return nil, err
	}
	if found := content.FindRRset(name, rrtype); found != nil {
		return found, nil
	}
	return nil, &APIError{
		Status:  http.StatusNotFound,
		Code:    "RRSET_NOT_FOUND",
		Message: "No s'ha trobat cap registre " + strings.ToUpper(rrtype) + " a " + name,
		Method:  http.MethodGet,
		URL:     c.base + "/api/v1/zones/" + url.PathEscape(zone) + "/rrsets",
	}
}

// PutRRset creates or updates an RRset.
func (c *DNSClient) PutRRset(ctx context.Context, zone, name, rrtype string, ttl uint32, values []string) (*RRset, error) {
	body := struct {
		TTL    uint32   `json:"ttl"`
		Values []string `json:"values"`
	}{TTL: ttl, Values: values}

	var out RRset
	err := c.retryOnConflict(ctx, func() error {
		return c.do(ctx, http.MethodPut, rrsetPath(zone, name, rrtype), body, &out)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteRRset deletes an RRset.
func (c *DNSClient) DeleteRRset(ctx context.Context, zone, name, rrtype string) error {
	return c.retryOnConflict(ctx, func() error {
		return c.do(ctx, http.MethodDelete, rrsetPath(zone, name, rrtype), nil, nil)
	})
}

func (c *DNSClient) retryOnConflict(ctx context.Context, fn func() error) error {
	err := fn()
	if !IsLostRace(err) {
		return err
	}
	select {
	case <-ctx.Done():
		return err
	case <-time.After(time.Second):
	}
	return fn()
}

func rrsetPath(zone, name, rrtype string) string {
	return "/api/v1/zones/" + url.PathEscape(zone) +
		"/rrsets/" + url.PathEscape(NormalizeOwnerName(name)) +
		"/" + url.PathEscape(strings.ToUpper(strings.TrimSpace(rrtype)))
}
