package cmd

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"nubulusttun/pkg/client"
	"nubulusttun/pkg/output"
)

var (
	exposeSubdomain      string
	exposeHostname       string
	exposeZone           string
	exposeTunnelID       string
	exposeUpstreamHost   string
	exposeUpstreamScheme string
	exposeLocal          bool
	exposePublicIP       bool
	exposeDetach         bool
	exposeTTL            uint32
)

var adjectives = []string{"swift", "bright", "cool", "fast", "brave", "calm", "keen", "epic", "hyper", "super"}
var nouns = []string{"fox", "wave", "wolf", "node", "link", "hawk", "lion", "flux", "core", "spark"}

func generateRandomSubdomain() string {
	adjIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(adjectives))))
	nounIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(nouns))))
	num, _ := rand.Int(rand.Reader, big.NewInt(99))
	return fmt.Sprintf("%s-%s-%02d", adjectives[adjIdx.Int64()], nouns[nounIdx.Int64()], num.Int64()+1)
}

var exposeCmd = &cobra.Command{
	Use:     "expose <port>",
	Aliases: []string{"share", "forward"},
	Short:   "Exposar un port a internet a través d'un túnel i DNS (estil ngrok)",
	Long: `Exposa un servidor local a internet automàticament utilitzant la IP interna de la xarxa local.

Aquesta ordre realitza de manera automàtica:
1. Detecció automàtica de la IP de xarxa local (LAN IP, ex. 192.168.1.27) com a destinació upstream.
2. Resolució o selecció del túnel WireGuard actiu.
3. Generació o assignació d'un subdomini a la zona DNS (ex. myapp.tun.aleix.cloud).
4. Creació del registre CNAME al DNS de Nubulus cap al CNAME Target del túnel.
5. Creació de la ruta de trànsit al túnel cap a la IP local (IP_LAN:port).
6. En mode primer pla (per defecte), monitoritza el túnel i en prémer Ctrl+C neteja
   automàticament la ruta i el registre DNS.

Exemples:
  # Exposar el port 9002 (utilitzant la IP LAN local, ex. 192.168.1.27)
  nubulus expose 9002

  # Exposar amb subdomini personalitzat (ex. gloria.tun.aleix.cloud)
  nubulus expose 9002 -s gloria

  # Forçar l'ús de 127.0.0.1 (per a túnels connectats a la mateixa màquina)
  nubulus expose 9002 --local

  # Forçar l'ús de la IP pública externa
  nubulus expose 9002 --public-ip

  # Exposar amb hostname complet i deixar-ho permanent (--detach)
  nubulus expose 9002 -H app.tun.aleix.cloud --detach
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rawTarget := args[0]

		scheme := exposeUpstreamScheme
		host := exposeUpstreamHost
		port := 0

		// Parse raw target if provided as URL or host:port or port
		if strings.Contains(rawTarget, "://") {
			u, err := url.Parse(rawTarget)
			if err != nil {
				return fmt.Errorf("URL invàlida: %w", err)
			}
			scheme = u.Scheme
			host = u.Hostname()
			if u.Port() != "" {
				p, _ := strconv.Atoi(u.Port())
				port = p
			}
		} else if strings.Contains(rawTarget, ":") {
			parts := strings.Split(rawTarget, ":")
			host = parts[0]
			p, err := strconv.Atoi(parts[1])
			if err != nil {
				return fmt.Errorf("port invàlid: %w", err)
			}
			port = p
		} else {
			p, err := strconv.Atoi(rawTarget)
			if err != nil {
				return fmt.Errorf("port invàlid '%s': especifiqueu un número de port com 9002", rawTarget)
			}
			port = p
		}

		// Resolve host: default to local LAN IP (e.g. 192.168.1.27) unless --local or --public-ip or explicit --upstream-host is provided
		if cmd.Flags().Changed("upstream-host") && host != "" {
			// keep user provided host
		} else if exposeLocal {
			host = "127.0.0.1"
		} else if exposePublicIP {
			pubIP, err := client.GetPublicIP(cmd.Context())
			if err == nil && pubIP != "" {
				host = pubIP
			} else {
				return fmt.Errorf("no s'ha pogut determinar la IP pública: %w", err)
			}
		} else if host == "" || host == "localhost" || host == "127.0.0.1" {
			lanIP, err := client.GetLocalLANIP()
			if err == nil && lanIP != "" {
				host = lanIP
			} else {
				host = "127.0.0.1"
			}
		}

		if port <= 0 || port > 65535 {
			return fmt.Errorf("el port ha d'estar entre 1 i 65535")
		}

		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		// 1. Resoldre Túnel
		selectedTunnelID := exposeTunnelID
		if selectedTunnelID == "" {
			selectedTunnelID = defaultTunnel
		}

		if selectedTunnelID == "" {
			tunnels, err := c.Tunnel.ListTunnels(cmd.Context())
			if err != nil {
				return fmt.Errorf("error llistant túnels per autoselecció: %w", err)
			}
			if len(tunnels) == 0 {
				return fmt.Errorf("no s'ha trobat cap túnel al compte. Creeu-ne un amb 'nubulus tunnels create'")
			}
			if len(tunnels) == 1 {
				selectedTunnelID = tunnels[0].ID
			} else {
				// Search if any tunnel has external_id or name matching
				var listMsg []string
				for _, t := range tunnels {
					listMsg = append(listMsg, fmt.Sprintf("  • ID: %s (nom: %s, subdomini: %s)", t.ID, t.Name, t.TunnelSubdomain))
				}
				return fmt.Errorf("s'han trobat múltiples túnels al compte. Especifiqueu quin voleu usar amb --tunnel <id> o a la configuració default_tunnel:\n%s", strings.Join(listMsg, "\n"))
			}
		}

		tunnelDetail, err := c.Tunnel.GetTunnel(cmd.Context(), selectedTunnelID)
		if err != nil {
			return fmt.Errorf("error obtenint detall del túnel '%s': %w", selectedTunnelID, err)
		}
		targetTunnel := tunnelDetail.Tunnel

		// 2. Resoldre Zona DNS
		targetZone := exposeZone
		if targetZone == "" {
			targetZone = defaultZone
		}
		if targetZone == "" {
			// Fallback: try "tun.aleix.cloud" or first available active zone
			zones, err := c.DNS.ListZones(cmd.Context())
			if err == nil {
				for _, z := range zones {
					if z.Name == "tun.aleix.cloud" && z.Status == "active" {
						targetZone = "tun.aleix.cloud"
						break
					}
				}
				if targetZone == "" && len(zones) > 0 {
					targetZone = zones[0].Name
				}
			}
		}
		if targetZone == "" {
			return fmt.Errorf("cal especificar la zona DNS amb --zone <zona> (o configurar default_zone al config)")
		}

		// 3. Resoldre Hostname i Subdomini
		var fqdn string
		var subLabel string

		if exposeHostname != "" {
			fqdn = strings.ToLower(strings.TrimSpace(exposeHostname))
			if !strings.HasSuffix(fqdn, "."+targetZone) && fqdn != targetZone {
				return fmt.Errorf("el hostname '%s' no pertany a la zona '%s'", fqdn, targetZone)
			}
			subLabel = strings.TrimSuffix(strings.TrimSuffix(fqdn, "."+targetZone), ".")
		} else if exposeSubdomain != "" {
			subLabel = strings.ToLower(strings.TrimSpace(exposeSubdomain))
			fqdn = fmt.Sprintf("%s.%s", subLabel, targetZone)
		} else {
			subLabel = generateRandomSubdomain()
			fqdn = fmt.Sprintf("%s.%s", subLabel, targetZone)
		}

		cnameValue := targetTunnel.TunnelSubdomain
		if !strings.HasSuffix(cnameValue, ".") {
			cnameValue += "."
		}

		fmt.Println("Configurant exposició pública...")
		fmt.Printf("• Túnel:     %s (%s)\n", targetTunnel.ID, targetTunnel.TunnelSubdomain)
		fmt.Printf("• Zona DNS:  %s\n", targetZone)
		fmt.Printf("• Hostname:  %s\n", fqdn)
		fmt.Printf("• Upstream:  %s://%s:%d\n\n", scheme, host, port)

		// 4. Crear o actualitzar registre CNAME al DNS
		fmt.Printf("1/2 Creant registre DNS CNAME '%s' -> '%s' (TTL: %d)... ", subLabel, cnameValue, exposeTTL)
		qualifiedCNAME, ok := client.QualifyName(subLabel, targetZone)
		if !ok {
			return fmt.Errorf("el nom '%s' és invàlid per a la zona '%s'", subLabel, targetZone)
		}

		_, err = c.DNS.PutRRset(cmd.Context(), targetZone, qualifiedCNAME, "CNAME", exposeTTL, []string{cnameValue})
		if err != nil {
			fmt.Println("FALLAT")
			return fmt.Errorf("error creant registre DNS CNAME: %w", err)
		}
		fmt.Println("OK")

		// 5. Crear la ruta al túnel
		fmt.Printf("2/2 Creant ruta de túnel per a '%s'... ", fqdn)
		createdRoute, err := c.Tunnel.CreateRoute(cmd.Context(), targetTunnel.ID, client.CreateRouteInput{
			Type:           "host",
			Hostname:       fqdn,
			PathPrefix:     "/",
			UpstreamHost:   host,
			UpstreamPort:   port,
			UpstreamScheme: scheme,
			StripPrefix:    false,
			Priority:       100,
		})
		if err != nil {
			fmt.Println("FALLAT")
			// Attempt to cleanup DNS CNAME on route failure
			_ = c.DNS.DeleteRRset(cmd.Context(), targetZone, qualifiedCNAME, "CNAME")
			return fmt.Errorf("error creant ruta de túnel: %w", err)
		}
		fmt.Println("OK")

		publicURL := fmt.Sprintf("https://%s", fqdn)
		forwardTarget := fmt.Sprintf("%s://%s:%d", scheme, host, port)

		if exposeDetach {
			output.Success("\nServidor exposat permanentment amb èxit!")
			fmt.Printf("  • URL Pública: %s\n", publicURL)
			fmt.Printf("  • Encaminat a: %s\n", forwardTarget)
			fmt.Printf("  • ID Ruta:     %s\n", createdRoute.ID)
			fmt.Println("\nPer alliberar-lo manualment:")
			fmt.Printf("  nubulus routes delete %s %s\n", targetTunnel.ID, createdRoute.ID)
			fmt.Printf("  nubulus records delete %s %s CNAME\n", targetZone, subLabel)
			return nil
		}

		// Foreground monitoring mode (ngrok-like)
		fmt.Println()
		printBanner(publicURL, forwardTarget, targetTunnel.ID, targetTunnel.OnlineStatus, fqdn)

		// Wait for SIGINT or SIGTERM
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Heartbeat ticker to show connection alive
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sigChan:
				fmt.Println("\n\nAturant exposició i netejant recursos...")

				// Cleanup route
				fmt.Printf("• Eliminant ruta de túnel '%s'... ", createdRoute.ID)
				if err := c.Tunnel.DeleteRoute(cmd.Context(), targetTunnel.ID, createdRoute.ID); err != nil {
					fmt.Printf("error: %v\n", err)
				} else {
					fmt.Println("OK")
				}

				// Cleanup DNS CNAME
				fmt.Printf("• Eliminant registre DNS CNAME '%s'... ", qualifiedCNAME)
				if err := c.DNS.DeleteRRset(cmd.Context(), targetZone, qualifiedCNAME, "CNAME"); err != nil {
					fmt.Printf("error: %v\n", err)
				} else {
					fmt.Println("OK")
				}

				output.Success("Recursos alliberats correctament. Adeu!")
				return nil

			case <-ticker.C:
				// Optional status check
				tCheck, err := c.Tunnel.GetTunnel(cmd.Context(), targetTunnel.ID)
				if err == nil && tCheck.Tunnel != nil {
					_ = tCheck.Tunnel.OnlineStatus
				}
			}
		}
	},
}

func printBanner(publicURL, forwardTarget, tunnelID, onlineStatus, fqdn string) {
	width := 68
	border := strings.Repeat("─", width)

	fmt.Printf("┌%s┐\n", border)
	fmt.Printf("│ %-66s │\n", "🚀 NUBULUS TUNNEL EXPOSE (ngrok mode)")
	fmt.Printf("├%s┤\n", border)
	fmt.Printf("│  URL Pública:       %-46s │\n", publicURL)
	fmt.Printf("│  Encaminat cap a:   %-46s │\n", forwardTarget)
	fmt.Printf("│  ID del Túnel:      %-46s │\n", tunnelID)
	fmt.Printf("│  Estat Connexió:    %-46s │\n", onlineStatus)
	fmt.Printf("├%s┤\n", border)
	fmt.Printf("│  %s │\n", "Premeu Ctrl+C per finalitzar i alliberar el DNS i la ruta.")
	fmt.Printf("└%s┘\n", border)
}

func init() {
	exposeCmd.Flags().StringVarP(&exposeSubdomain, "subdomain", "s", "", "Subdomini prefix (ex. 'api' o 'demo')")
	exposeCmd.Flags().StringVarP(&exposeHostname, "hostname", "H", "", "Hostname FQDN complet (ex. 'demo.tun.aleix.cloud')")
	exposeCmd.Flags().StringVarP(&exposeZone, "zone", "z", "", "Zona DNS on crear el registre CNAME (per defecte config default_zone)")
	exposeCmd.Flags().StringVarP(&exposeTunnelID, "tunnel", "t", "", "ID del túnel WireGuard (per defecte config default_tunnel)")
	exposeCmd.Flags().StringVar(&exposeUpstreamHost, "upstream-host", "", "Host/IP de l'aplicació (per defecte: IP LAN local detectada)")
	exposeCmd.Flags().BoolVar(&exposeLocal, "local", false, "Utilitzar 127.0.0.1 (localhost) com a host upstream")
	exposeCmd.Flags().BoolVar(&exposePublicIP, "public-ip", false, "Utilitzar la IP pública externa com a host upstream")
	exposeCmd.Flags().StringVar(&exposeUpstreamScheme, "upstream-scheme", "http", "Esquema: http o https")
	exposeCmd.Flags().BoolVarP(&exposeDetach, "detach", "d", false, "Executar en segon pla (permanent) sense neteja automàtica al sortir")
	exposeCmd.Flags().Uint32Var(&exposeTTL, "ttl", 60, "TTL del registre DNS CNAME")

	rootCmd.AddCommand(exposeCmd)
}
