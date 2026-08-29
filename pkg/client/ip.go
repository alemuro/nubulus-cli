package client

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// GetLocalLANIP finds the preferred outbound local LAN IP (e.g. 192.168.x.x, 10.x.x.x)
// by probing the routing table via a UDP connection.
func GetLocalLANIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		if !localAddr.IP.IsLoopback() && localAddr.IP.To4() != nil {
			return localAddr.IP.String(), nil
		}
	}

	// Fallback: iterate over non-loopback network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// Skip docker/veth bridges if possible
		if strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "veth") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no s'ha pogut detectar la IP interna de la xarxa local")
}

// GetPublicIP resolves the external/public IPv4 address of the local machine.
func GetPublicIP(ctx context.Context) (string, error) {
	endpoints := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}

	httpClient := &http.Client{Timeout: 3 * time.Second}

	for _, ep := range endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "curl/8.0")

		resp, err := httpClient.Do(req)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		if err == nil && resp.StatusCode == http.StatusOK {
			ip := strings.TrimSpace(string(body))
			if ip != "" && !strings.Contains(ip, "<") && !strings.Contains(ip, " ") {
				return ip, nil
			}
		}
	}

	return "", fmt.Errorf("no s'ha pogut resoldre la IP pública automàticament")
}
