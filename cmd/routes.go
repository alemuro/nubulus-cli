package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alemuro/nubulus-cli/pkg/client"
	"github.com/alemuro/nubulus-cli/pkg/output"
)

var (
	routeType           string
	routeHostname       string
	routePathPrefix     string
	routeUpstreamHost   string
	routeUpstreamPort   int
	routeUpstreamScheme string
	routeStripPrefix    bool
	routePriority       int
	deleteRouteForce    bool

	// Update flags
	updateUpstreamHost   string
	updateUpstreamPort   int
	updateUpstreamScheme string
	updateStripPrefix    string // "true" or "false"
	updatePriority       int
	updateEnabled        string // "true" or "false"
)

var routesCmd = &cobra.Command{
	Use:     "routes",
	Aliases: []string{"route", "r"},
	Short:   "Administrar rutes de túnels",
	Long:    "Ordres per a llistar, crear, obtenir detalls, modificar i eliminar rutes de trànsit associades a un túnel.",
}

var routesListCmd = &cobra.Command{
	Use:     "list <tunnel-id>",
	Aliases: []string{"ls"},
	Short:   "Llistar totes les rutes associades a un túnel",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelID := args[0]
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		routes, err := c.Tunnel.ListRoutes(cmd.Context(), tunnelID)
		if err != nil {
			return err
		}

		return output.PrintStructured(os.Stdout, output.Format(outputFormat), routes, func(w io.Writer) error {
			if len(routes) == 0 {
				fmt.Fprintf(w, "El túnel '%s' no té cap ruta configurada.\n", tunnelID)
				return nil
			}

			tw := output.NewTableWriter(w)
			fmt.Fprintln(tw, "ID RUTA\tHOSTNAME\tTIPUS\tPREFIX\tDESTINACIÓ UPSTREAM\tSTRIP\tACTIVA\tPRIORITAT")
			for _, r := range routes {
				upstream := fmt.Sprintf("%s://%s:%d", r.UpstreamScheme, r.UpstreamHost, r.UpstreamPort)
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%t\t%t\t%d\n",
					r.ID,
					r.Hostname,
					r.Type,
					r.PathPrefix,
					upstream,
					r.StripPrefix,
					r.Enabled,
					r.Priority,
				)
			}
			return tw.Flush()
		})
	},
}

var routesGetCmd = &cobra.Command{
	Use:   "get <tunnel-id> <route-id>",
	Short: "Obtenir informació detallada d'una ruta concreta",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelID := args[0]
		routeID := args[1]

		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		route, err := c.Tunnel.GetRoute(cmd.Context(), tunnelID, routeID)
		if err != nil {
			return err
		}

		return output.PrintStructured(os.Stdout, output.Format(outputFormat), route, func(w io.Writer) error {
			tw := output.NewTableWriter(w)
			fmt.Fprintf(tw, "ID Ruta:\t%s\n", route.ID)
			fmt.Fprintf(tw, "ID Túnel:\t%s\n", route.TunnelID)
			fmt.Fprintf(tw, "Hostname:\t%s\n", route.Hostname)
			fmt.Fprintf(tw, "Tipus:\t%s\n", route.Type)
			fmt.Fprintf(tw, "Prefix de Ruta:\t%s\n", route.PathPrefix)
			fmt.Fprintf(tw, "Upstream Host:\t%s\n", route.UpstreamHost)
			fmt.Fprintf(tw, "Upstream Port:\t%d\n", route.UpstreamPort)
			fmt.Fprintf(tw, "Upstream Scheme:\t%s\n", route.UpstreamScheme)
			fmt.Fprintf(tw, "Strip Prefix:\t%t\n", route.StripPrefix)
			fmt.Fprintf(tw, "Activa (Enabled):\t%t\n", route.Enabled)
			fmt.Fprintf(tw, "Prioritat:\t%d\n", route.Priority)
			fmt.Fprintf(tw, "Creat:\t%s\n", route.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Fprintf(tw, "Actualitzat:\t%s\n", route.UpdatedAt.Format("2006-01-02 15:04:05"))
			return tw.Flush()
		})
	},
}

var routesCreateCmd = &cobra.Command{
	Use:   "create <tunnel-id>",
	Short: "Crear una nova ruta en un túnel",
	Long: `Afegeix una ruta que redirigeix les peticions per a un hostname públic
a través del túnel WireGuard cap al servei amfitrió o IP interna indicada.

Exemples:
  # Ruta de host completa (ex. app.example.com -> http://127.0.0.1:8080)
  nubulus routes create tun_123 --hostname app.example.com --upstream-host 127.0.0.1 --upstream-port 8080

  # Ruta per prefix (ex. api.example.com/v1 -> http://10.0.0.5:3000 eliminant el prefix)
  nubulus routes create tun_123 --type path --hostname api.example.com --path-prefix /v1 --upstream-host 10.0.0.5 --upstream-port 3000 --strip-prefix
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelID := args[0]

		if routeHostname == "" {
			return fmt.Errorf("cal especificar --hostname")
		}
		if routeUpstreamHost == "" {
			return fmt.Errorf("cal especificar --upstream-host")
		}
		if routeUpstreamPort <= 0 || routeUpstreamPort > 65535 {
			return fmt.Errorf("--upstream-port ha d'estar entre 1 i 65535")
		}

		if routeType == "path" {
			if routePathPrefix == "" || routePathPrefix == "/" {
				return fmt.Errorf("per a rutes de tipus 'path' cal especificar un --path-prefix distint de '/' (ex. /api)")
			}
			if !strings.HasPrefix(routePathPrefix, "/") {
				return fmt.Errorf("--path-prefix ha de començar per '/'")
			}
		} else {
			routePathPrefix = "/"
		}

		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		created, err := c.Tunnel.CreateRoute(cmd.Context(), tunnelID, client.CreateRouteInput{
			Type:           routeType,
			Hostname:       routeHostname,
			PathPrefix:     routePathPrefix,
			UpstreamHost:   routeUpstreamHost,
			UpstreamPort:   routeUpstreamPort,
			UpstreamScheme: routeUpstreamScheme,
			StripPrefix:    routeStripPrefix,
			Priority:       routePriority,
		})
		if err != nil {
			return err
		}

		// If priority is 0, we need an explicit update because create defaults 0 to 100
		if routePriority == 0 && created.Priority != 0 {
			zero := 0
			updated, err := c.Tunnel.UpdateRoute(cmd.Context(), tunnelID, created.ID, client.UpdateRouteInput{
				Priority: &zero,
			})
			if err == nil {
				created = updated
			}
		}

		if outputFormat == "json" || outputFormat == "yaml" {
			return output.PrintStructured(os.Stdout, output.Format(outputFormat), created, nil)
		}

		output.Success("Ruta creada amb èxit (ID: %s) per a '%s'", created.ID, created.Hostname)
		fmt.Printf("• Trànsit encaminat a: %s://%s:%d\n", created.UpstreamScheme, created.UpstreamHost, created.UpstreamPort)
		fmt.Println("Recordeu configurar el registre CNAME al vostre DNS perquè apunti al túnel!")
		return nil
	},
}

var routesUpdateCmd = &cobra.Command{
	Use:   "update <tunnel-id> <route-id>",
	Short: "Actualitzar paràmetres d'una ruta existent",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelID := args[0]
		routeID := args[1]

		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		var updateInput client.UpdateRouteInput
		hasChanges := false

		if cmd.Flags().Changed("upstream-host") {
			updateInput.UpstreamHost = &updateUpstreamHost
			hasChanges = true
		}
		if cmd.Flags().Changed("upstream-port") {
			if updateUpstreamPort <= 0 || updateUpstreamPort > 65535 {
				return fmt.Errorf("--upstream-port ha d'estar entre 1 i 65535")
			}
			updateInput.UpstreamPort = &updateUpstreamPort
			hasChanges = true
		}
		if cmd.Flags().Changed("upstream-scheme") {
			updateInput.UpstreamScheme = &updateUpstreamScheme
			hasChanges = true
		}
		if cmd.Flags().Changed("strip-prefix") {
			v := strings.ToLower(updateStripPrefix) == "true" || updateStripPrefix == "1"
			updateInput.StripPrefix = &v
			hasChanges = true
		}
		if cmd.Flags().Changed("priority") {
			updateInput.Priority = &updatePriority
			hasChanges = true
		}
		if cmd.Flags().Changed("enabled") {
			v := strings.ToLower(updateEnabled) == "true" || updateEnabled == "1"
			updateInput.Enabled = &v
			hasChanges = true
		}

		if !hasChanges {
			return fmt.Errorf("cal especificar com a mínim un camp per a actualitzar")
		}

		updated, err := c.Tunnel.UpdateRoute(cmd.Context(), tunnelID, routeID, updateInput)
		if err != nil {
			return err
		}

		if outputFormat == "json" || outputFormat == "yaml" {
			return output.PrintStructured(os.Stdout, output.Format(outputFormat), updated, nil)
		}

		output.Success("Ruta '%s' actualitzada correctament.", routeID)
		return nil
	},
}

var routesDeleteCmd = &cobra.Command{
	Use:   "delete <tunnel-id> <route-id>",
	Short: "Eliminar una ruta",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelID := args[0]
		routeID := args[1]

		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		if !deleteRouteForce {
			fmt.Printf("Esteu segur que voleu eliminar la ruta '%s' del túnel '%s'? [s/N]: ", routeID, tunnelID)
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "s" && strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "si" {
				fmt.Println("Operació cancel·lada.")
				return nil
			}
		}

		if err := c.Tunnel.DeleteRoute(cmd.Context(), tunnelID, routeID); err != nil {
			return err
		}

		output.Success("Ruta '%s' eliminada correctament.", routeID)
		return nil
	},
}

func init() {
	routesCreateCmd.Flags().StringVar(&routeType, "type", "host", "Tipus de ruta: host o path")
	routesCreateCmd.Flags().StringVar(&routeHostname, "hostname", "", "Nom de domini FQDN a servir (obligatori)")
	routesCreateCmd.Flags().StringVar(&routePathPrefix, "path-prefix", "/", "Prefix del camí (obligatori si type=path)")
	routesCreateCmd.Flags().StringVar(&routeUpstreamHost, "upstream-host", "", "Host o IP intern cap a on encaminar (obligatori)")
	routesCreateCmd.Flags().IntVar(&routeUpstreamPort, "upstream-port", 80, "Port del servei intern (obligatori)")
	routesCreateCmd.Flags().StringVar(&routeUpstreamScheme, "upstream-scheme", "http", "Esquema: http o https")
	routesCreateCmd.Flags().BoolVar(&routeStripPrefix, "strip-prefix", false, "Eliminar el path-prefix abans d'arribar a l'upstream")
	routesCreateCmd.Flags().IntVar(&routePriority, "priority", 100, "Prioritat de la ruta (menor número guanya)")

	routesUpdateCmd.Flags().StringVar(&updateUpstreamHost, "upstream-host", "", "Nou host o IP intern")
	routesUpdateCmd.Flags().IntVar(&updateUpstreamPort, "upstream-port", 0, "Nou port intern")
	routesUpdateCmd.Flags().StringVar(&updateUpstreamScheme, "upstream-scheme", "", "Nou esquema (http/https)")
	routesUpdateCmd.Flags().StringVar(&updateStripPrefix, "strip-prefix", "", "Eliminar prefix (true/false)")
	routesUpdateCmd.Flags().IntVar(&updatePriority, "priority", 100, "Nova prioritat")
	routesUpdateCmd.Flags().StringVar(&updateEnabled, "enabled", "", "Habilitar o inhabilitar la ruta (true/false)")

	routesDeleteCmd.Flags().BoolVarP(&deleteRouteForce, "yes", "y", false, "Eliminar sense demanar confirmació interactiva")

	routesCmd.AddCommand(routesListCmd)
	routesCmd.AddCommand(routesGetCmd)
	routesCmd.AddCommand(routesCreateCmd)
	routesCmd.AddCommand(routesUpdateCmd)
	routesCmd.AddCommand(routesDeleteCmd)

	rootCmd.AddCommand(routesCmd)
}
