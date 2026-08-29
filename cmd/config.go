package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/alemuro/nubulus-cli/pkg/client"
	"github.com/alemuro/nubulus-cli/pkg/output"
)

var (
	configForceOverwrite bool
	initClientID         string
	initClientSecret     string
	initDefaultZone      string
	initDefaultTunnel    string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gestionar la configuració de la CLI",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Mostrar la configuració resolta actual",
	Run: func(cmd *cobra.Command, args []string) {
		maskedSecret := ""
		if len(clientSecret) > 6 {
			maskedSecret = clientSecret[:3] + "..." + clientSecret[len(clientSecret)-3:]
		} else if clientSecret != "" {
			maskedSecret = "******"
		}

		tw := output.NewTableWriter(os.Stdout)
		fmt.Fprintf(tw, "Client ID:\t%s\n", clientID)
		fmt.Fprintf(tw, "Client Secret:\t%s\n", maskedSecret)
		fmt.Fprintf(tw, "Default Zone:\t%s\n", defaultZone)
		fmt.Fprintf(tw, "Default Tunnel:\t%s\n", defaultTunnel)
		fmt.Fprintf(tw, "Token URL:\t%s\n", tokenURL)
		fmt.Fprintf(tw, "Project ID:\t%s\n", projectID)
		fmt.Fprintf(tw, "DNS Endpoint:\t%s\n", dnsEndpoint)
		fmt.Fprintf(tw, "Tunnel Endpoint:\t%s\n", tunnelEndpoint)
		_ = tw.Flush()
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Configurar interactivament les credencials (~/.config/nubulus/config.yaml)",
	Long: `Demanarà interactivament el Client ID i el Client Secret d'un Application Token
de Nubulus Cloud, a més de la zona DNS i túnel per defecte (opcionals), i desarà la configuració
a ~/.config/nubulus/config.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir := filepath.Join(os.Getenv("HOME"), ".config", "nubulus")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("error creant directori de configuració: %w", err)
		}

		targetFile := filepath.Join(configDir, "config.yaml")
		if _, err := os.Stat(targetFile); err == nil && !configForceOverwrite {
			fmt.Printf("El fitxer '%s' ja existeix. Voleu sobreescriure'l? [s/N]: ", targetFile)
			reader := bufio.NewReader(os.Stdin)
			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))
			if confirm != "s" && confirm != "y" && confirm != "si" {
				fmt.Println("Operació cancel·lada.")
				return nil
			}
		}

		reader := bufio.NewReader(os.Stdin)

		// 1. Demanar Client ID
		cID := initClientID
		if cID == "" {
			fmt.Print("Introduïu el Client ID: ")
			line, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			cID = strings.TrimSpace(line)
			if cID == "" {
				return fmt.Errorf("el Client ID no pot estar buit")
			}
		}

		// 2. Demanar Client Secret
		cSecret := initClientSecret
		if cSecret == "" {
			fmt.Print("Introduïu el Client Secret: ")
			if term.IsTerminal(int(os.Stdin.Fd())) {
				byteSecret, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println() // salt de línia després d'ocultar l'entrada
				if err != nil {
					return fmt.Errorf("error llegint secret: %w", err)
				}
				cSecret = strings.TrimSpace(string(byteSecret))
			} else {
				line, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				cSecret = strings.TrimSpace(line)
			}

			if cSecret == "" {
				return fmt.Errorf("el Client Secret no pot estar buit")
			}
		}

		// 3. Demanar Zona DNS per defecte (opcional)
		dZone := initDefaultZone
		if dZone == "" {
			fmt.Print("Zona DNS per defecte (opcional, ex. tun.aleix.cloud) [tun.aleix.cloud]: ")
			line, err := reader.ReadString('\n')
			if err == nil {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					dZone = trimmed
				} else {
					dZone = "tun.aleix.cloud"
				}
			}
		}

		// 4. Demanar Túnel per defecte (opcional)
		dTunnel := initDefaultTunnel
		if dTunnel == "" {
			fmt.Print("ID de Túnel per defecte (opcional, ex. cb781a20-708e-429a-9e2b-cf54e1e81d9d): ")
			line, err := reader.ReadString('\n')
			if err == nil {
				dTunnel = strings.TrimSpace(line)
			}
		}

		cfg := client.Config{
			ClientID:       cID,
			ClientSecret:   cSecret,
			DefaultZone:    dZone,
			DefaultTunnel:  dTunnel,
			TokenURL:       client.DefaultTokenURL,
			ProjectID:      client.DefaultProjectID,
			DNSEndpoint:    client.DefaultDNSEndpoint,
			TunnelEndpoint: client.DefaultTunnelEndpoint,
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("error codificant fitxer yaml: %w", err)
		}

		if err := os.WriteFile(targetFile, data, 0600); err != nil {
			return fmt.Errorf("error desant fitxer: %w", err)
		}

		output.Success("Configuració desada correctament a '%s' amb permisos 0600.", targetFile)
		fmt.Println("Ja podeu utilitzar les ordres de 'nubulus' (ex: nubulus expose 3000, nubulus zones list).")
		return nil
	},
}

func init() {
	configInitCmd.Flags().BoolVarP(&configForceOverwrite, "force", "f", false, "Sobreescriure el fitxer existent sense demanar confirmació")
	configInitCmd.Flags().StringVar(&initClientID, "client-id", "", "Client ID directe (sense mode interactiu)")
	configInitCmd.Flags().StringVar(&initClientSecret, "client-secret", "", "Client Secret directe (sense mode interactiu)")
	configInitCmd.Flags().StringVar(&initDefaultZone, "default-zone", "", "Zona DNS per defecte")
	configInitCmd.Flags().StringVar(&initDefaultTunnel, "default-tunnel", "", "ID de túnel per defecte")

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)

	rootCmd.AddCommand(configCmd)
}
