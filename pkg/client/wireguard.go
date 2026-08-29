package client

import "fmt"

// FormatWireGuardConfig produces standard WireGuard config (.conf) text.
func FormatWireGuardConfig(cfg WireGuardConfig) string {
	content := "[Interface]\n"
	if cfg.Interface.PrivateKey != "" {
		content += fmt.Sprintf("PrivateKey = %s\n", cfg.Interface.PrivateKey)
	}
	if cfg.Interface.Address != "" {
		content += fmt.Sprintf("Address = %s\n", cfg.Interface.Address)
	}
	if cfg.Interface.DNS != "" {
		content += fmt.Sprintf("DNS = %s\n", cfg.Interface.DNS)
	}

	content += "\n[Peer]\n"
	if cfg.Peer.PublicKey != "" {
		content += fmt.Sprintf("PublicKey = %s\n", cfg.Peer.PublicKey)
	}
	if cfg.Peer.Endpoint != "" {
		content += fmt.Sprintf("Endpoint = %s\n", cfg.Peer.Endpoint)
	}
	if cfg.Peer.AllowedIPs != "" {
		content += fmt.Sprintf("AllowedIPs = %s\n", cfg.Peer.AllowedIPs)
	}
	if cfg.Peer.PersistentKeepalive > 0 {
		content += fmt.Sprintf("PersistentKeepalive = %d\n", cfg.Peer.PersistentKeepalive)
	}

	return content
}
