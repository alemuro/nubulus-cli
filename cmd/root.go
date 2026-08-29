package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/alemuro/nubulus-cli/pkg/client"
	"github.com/alemuro/nubulus-cli/pkg/output"
)

var (
	cfgFile        string
	clientID       string
	clientSecret   string
	tokenURL       string
	projectID      string
	dnsEndpoint    string
	tunnelEndpoint string
	defaultZone    string
	defaultTunnel  string
	outputFormat   string

	rootCmd = &cobra.Command{
		Use:   "nubulus",
		Short: "CLI per a interactuar amb les APIs de DNS i Túnels de Nubulus Cloud",
		Long: `nubulus és una eina de línia d'ordres per a administrar zones DNS,
registres (RRsets), túnels WireGuard i rutes a la plataforma Nubulus Cloud.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		output.Error("%s", client.FriendlyExplanation("executar l'ordre", err))
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Fitxer de configuració (per defecte ~/.config/nubulus/config.yaml o ./.nubulus.yaml)")
	rootCmd.PersistentFlags().StringVar(&clientID, "client-id", "", "OAuth2 Client ID (també variable NUBULUS_CLIENT_ID)")
	rootCmd.PersistentFlags().StringVar(&clientSecret, "client-secret", "", "OAuth2 Client Secret (també variable NUBULUS_CLIENT_SECRET)")
	rootCmd.PersistentFlags().StringVar(&tokenURL, "token-url", "", "URL d'obtenció de token OAuth2")
	rootCmd.PersistentFlags().StringVar(&projectID, "project-id", "", "ID de projecte de Nubulus")
	rootCmd.PersistentFlags().StringVar(&dnsEndpoint, "dns-endpoint", "", "Endpoint base de l'API DNS")
	rootCmd.PersistentFlags().StringVar(&tunnelEndpoint, "tunnel-endpoint", "", "Endpoint base de l'API de Túnels")
	rootCmd.PersistentFlags().StringVar(&defaultZone, "default-zone", "", "Zona DNS per defecte (també variable NUBULUS_DEFAULT_ZONE)")
	rootCmd.PersistentFlags().StringVar(&defaultTunnel, "default-tunnel", "", "ID de túnel per defecte (també variable NUBULUS_DEFAULT_TUNNEL)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Format de sortida: table, json, yaml")
}

func initConfig() {
	// Try loading config file if available
	configFileToLoad := cfgFile
	if configFileToLoad == "" {
		candidates := []string{
			filepath.Join(os.Getenv("HOME"), ".config", "nubulus", "config.yaml"),
			filepath.Join(os.Getenv("HOME"), ".nubulus.yaml"),
			".nubulus.yaml",
		}
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				configFileToLoad = path
				break
			}
		}
	}

	if configFileToLoad != "" {
		data, err := os.ReadFile(configFileToLoad)
		if err == nil {
			var fileCfg client.Config
			if err := yaml.Unmarshal(data, &fileCfg); err == nil {
				if clientID == "" && fileCfg.ClientID != "" {
					clientID = fileCfg.ClientID
				}
				if clientSecret == "" && fileCfg.ClientSecret != "" {
					clientSecret = fileCfg.ClientSecret
				}
				if tokenURL == "" && fileCfg.TokenURL != "" {
					tokenURL = fileCfg.TokenURL
				}
				if projectID == "" && fileCfg.ProjectID != "" {
					projectID = fileCfg.ProjectID
				}
				if dnsEndpoint == "" && fileCfg.DNSEndpoint != "" {
					dnsEndpoint = fileCfg.DNSEndpoint
				}
				if tunnelEndpoint == "" && fileCfg.TunnelEndpoint != "" {
					tunnelEndpoint = fileCfg.TunnelEndpoint
				}
				if defaultZone == "" && fileCfg.DefaultZone != "" {
					defaultZone = fileCfg.DefaultZone
				}
				if defaultTunnel == "" && fileCfg.DefaultTunnel != "" {
					defaultTunnel = fileCfg.DefaultTunnel
				}
			}
		}
	}

	// Environment variable fallback
	if clientID == "" {
		clientID = os.Getenv("NUBULUS_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("NUBULUS_CLIENT_SECRET")
	}
	if tokenURL == "" {
		tokenURL = os.Getenv("NUBULUS_TOKEN_URL")
	}
	if projectID == "" {
		projectID = os.Getenv("NUBULUS_PROJECT_ID")
	}
	if dnsEndpoint == "" {
		dnsEndpoint = os.Getenv("NUBULUS_DNS_ENDPOINT")
	}
	if tunnelEndpoint == "" {
		tunnelEndpoint = os.Getenv("NUBULUS_TUNNEL_ENDPOINT")
	}
	if defaultZone == "" {
		defaultZone = os.Getenv("NUBULUS_DEFAULT_ZONE")
	}
	if defaultTunnel == "" {
		defaultTunnel = os.Getenv("NUBULUS_DEFAULT_TUNNEL")
	}
}

func getClient(ctx context.Context) (*client.Client, error) {
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("credencials no configurades: cal proporcionar client_id i client_secret (per flags, fitxer de configuració o variables d'entorn NUBULUS_CLIENT_ID / NUBULUS_CLIENT_SECRET)")
	}

	return client.New(ctx, client.Config{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		TokenURL:       tokenURL,
		ProjectID:      projectID,
		DNSEndpoint:    dnsEndpoint,
		TunnelEndpoint: tunnelEndpoint,
		UserAgent:      "nubulus-cli/1.0.0",
	})
}
