package client

import "net/http"

const (
	DefaultTokenURL       = "https://idp.nubulusnetwork.es/oauth/v2/token"
	DefaultDNSEndpoint    = "https://dns.api.nubulusnetwork.es"
	DefaultTunnelEndpoint = "https://tunel.api.nubulusnetwork.es"
	DefaultProjectID      = "385111705782321341"
)

// Scopes builds the exact OAuth2 scope list required by Zitadel and Nubulus APIs.
func Scopes(projectID string) []string {
	return []string{
		"openid",
		"urn:zitadel:iam:org:project:id:" + projectID + ":aud",
		"urn:zitadel:iam:user:resourceowner",
		"urn:zitadel:iam:org:projects:roles",
	}
}

// Config holds all parameters to build a Nubulus API client.
type Config struct {
	ClientID     string `yaml:"client_id" json:"client_id"`
	ClientSecret string `yaml:"client_secret" json:"client_secret"`
	TokenURL     string `yaml:"token_url" json:"token_url"`
	ProjectID    string `yaml:"project_id" json:"project_id"`

	DNSEndpoint    string `yaml:"dns_endpoint" json:"dns_endpoint"`
	TunnelEndpoint string `yaml:"tunnel_endpoint" json:"tunnel_endpoint"`

	DefaultZone   string `yaml:"default_zone,omitempty" json:"default_zone,omitempty"`
	DefaultTunnel string `yaml:"default_tunnel,omitempty" json:"default_tunnel,omitempty"`

	UserAgent string `yaml:"user_agent" json:"user_agent"`

	HTTPClient *http.Client `yaml:"-" json:"-"`
}
