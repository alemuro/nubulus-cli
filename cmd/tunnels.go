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
	tunnelName        string
	tunnelExternalID  string
	tunnelSaveWGPath  string
	deleteTunnelForce bool
	filterExternalID  string
)

var tunnelsCmd = &cobra.Command{
	Use:     "tunnels",
	Aliases: []string{"tunnel", "tun", "t"},
	Short:   "Administrar túnels WireGuard",
	Long:    "Ordres per a llistar, crear, obtenir detalls, rotar credencials i eliminar túnels WireGuard a Nubulus Cloud.",
}

var tunnelsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Llistar tots els túnels del compte",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		var tunnels []client.TunnelSummary
		if filterExternalID != "" {
			tun, err := c.Tunnel.FindTunnelByExternalID(cmd.Context(), filterExternalID)
			if err != nil {
				return err
			}
			if tun != nil {
				tunnels = append(tunnels, *tun)
			}
		} else {
			all, err := c.Tunnel.ListTunnels(cmd.Context())
			if err != nil {
				return err
			}
			tunnels = all
		}

		return output.PrintStructured(os.Stdout, output.Format(outputFormat), tunnels, func(w io.Writer) error {
			if len(tunnels) == 0 {
				fmt.Fprintln(w, "No s'ha trobat cap túnel.")
				return nil
			}

			tw := output.NewTableWriter(w)
			fmt.Fprintln(tw, "ID\tNOM\tEXTERNAL ID\tSUBDOMINI TÚNEL\tIP WIREGUARD\tESTAT\tCONNEXIÓ\tRUTES")
			for _, t := range tunnels {
				name := t.Name
				if name == "" {
					name = "-"
				}
				extID := t.ExternalID
				if extID == "" {
					extID = "-"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\n",
					t.ID,
					name,
					extID,
					t.TunnelSubdomain,
					t.WireGuardIP,
					t.Status,
					t.OnlineStatus,
					t.RouteCount,
				)
			}
			return tw.Flush()
		})
	},
}

var tunnelsGetCmd = &cobra.Command{
	Use:   "get <tunnel-id>",
	Short: "Obtenir informació detallada d'un túnel i les seves rutes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelID := args[0]
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		tunWithRoutes, err := c.Tunnel.GetTunnel(cmd.Context(), tunnelID)
		if err != nil {
			return err
		}

		return output.PrintStructured(os.Stdout, output.Format(outputFormat), tunWithRoutes, func(w io.Writer) error {
			t := tunWithRoutes.Tunnel
			tw := output.NewTableWriter(w)
			fmt.Fprintf(tw, "ID Túnel:\t%s\n", t.ID)
			if t.Name != "" {
				fmt.Fprintf(tw, "Nom:\t%s\n", t.Name)
			}
			if t.ExternalID != "" {
				fmt.Fprintf(tw, "External ID:\t%s\n", t.ExternalID)
			}
			fmt.Fprintf(tw, "Subdomini Túnel:\t%s\n", t.TunnelSubdomain)
			fmt.Fprintf(tw, "IP WireGuard:\t%s\n", t.WireGuardIP)
			if t.WireGuardPublicKey != "" {
				fmt.Fprintf(tw, "Clau Pública WG:\t%s\n", t.WireGuardPublicKey)
			}
			fmt.Fprintf(tw, "Estat Túnel:\t%s\n", t.Status)
			fmt.Fprintf(tw, "Estat Connexió:\t%s\n", t.OnlineStatus)
			if t.LastHandshakeAt != nil {
				fmt.Fprintf(tw, "Últim Handshake:\t%s\n", t.LastHandshakeAt.Format("2006-01-02 15:04:05"))
			}
			fmt.Fprintf(tw, "Creat:\t%s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Fprintf(tw, "Actualitzat:\t%s\n", t.UpdatedAt.Format("2006-01-02 15:04:05"))
			_ = tw.Flush()

			fmt.Fprintln(w, "\n--- Rutes Associades ---")
			if len(tunWithRoutes.Routes) == 0 {
				fmt.Fprintln(w, "Cap ruta configurada en aquest túnel.")
				return nil
			}

			rtTw := output.NewTableWriter(w)
			fmt.Fprintln(rtTw, "ID RUTA\tHOSTNAME\tTIPUS\tPREFIX\tDESTINACIÓ UPSTREAM\tSTRIP\tACTIVA\tPRIORITAT")
			for _, r := range tunWithRoutes.Routes {
				upstream := fmt.Sprintf("%s://%s:%d", r.UpstreamScheme, r.UpstreamHost, r.UpstreamPort)
				fmt.Fprintf(rtTw, "%s\t%s\t%s\t%s\t%s\t%t\t%t\t%d\n",
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
			return rtTw.Flush()
		})
	},
}

var tunnelsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Crear un nou túnel WireGuard (o adoptar un existent)",
	Long: `Crea un túnel WireGuard a Nubulus Cloud.

ATENCIÓ: La clau privada WireGuard i el token del túnel només es mostren un sol cop en aquesta resposta!
Utilitzeu el flag --save-wg <fitxer.conf> per desar directament la configuració llesta per a wg-quick.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		res, err := c.Tunnel.CreateTunnel(cmd.Context(), client.CreateTunnelInput{
			Name:       tunnelName,
			ExternalID: tunnelExternalID,
		})
		if err != nil {
			return err
		}

		if outputFormat == "json" || outputFormat == "yaml" {
			return output.PrintStructured(os.Stdout, output.Format(outputFormat), res, nil)
		}

		if res.Adopted {
			output.Warn("El túnel amb external_id '%s' ja existia i ha estat adoptat (ID: %s).", tunnelExternalID, res.TunnelID)
			fmt.Println("Les credencials existents es mantenen. Si necessiteu un token nou, executeu: nubulus tunnels rotate-token " + res.TunnelID)
			return nil
		}

		output.Success("Túnel creat amb èxit! (ID: %s)", res.TunnelID)
		fmt.Printf("\n• Subdomini assignat: %s\n", res.TunnelSubdomain)
		fmt.Printf("• CNAME Target:       %s\n", res.CNAMETarget)
		fmt.Printf("• IP WireGuard:       %s\n", res.WireGuardIP)
		fmt.Printf("• Token de Túnel:     %s\n", res.TunnelToken)

		wgConf := client.FormatWireGuardConfig(res.WireGuard)

		if tunnelSaveWGPath != "" {
			if err := os.WriteFile(tunnelSaveWGPath, []byte(wgConf), 0600); err != nil {
				output.Warn("No s'ha pogut desar el fitxer WireGuard: %v", err)
			} else {
				output.Success("Configuració WireGuard desada a '%s' amb permisos 0600!", tunnelSaveWGPath)
				fmt.Printf("Podeu aixecar el túnel amb: sudo wg-quick up %s\n", tunnelSaveWGPath)
			}
		} else {
			fmt.Println("\n--- Configuració WireGuard (wg0.conf) ---")
			fmt.Print(wgConf)
			fmt.Println("----------------------------------------")
			fmt.Println("Podeu desar aquesta configuració en un fitxer .conf i utilitzar wg-quick per connectar-vos.")
		}

		return nil
	},
}

var tunnelsRotateTokenCmd = &cobra.Command{
	Use:   "rotate-token <tunnel-id>",
	Short: "Rotar les credencials (tunnel_token) d'un túnel",
	Long:  "Genera un nou token d'autenticació per al túnel i invalida immediatament l'anterior.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelID := args[0]
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		res, err := c.Tunnel.RotateToken(cmd.Context(), tunnelID)
		if err != nil {
			return err
		}

		if outputFormat == "json" || outputFormat == "yaml" {
			return output.PrintStructured(os.Stdout, output.Format(outputFormat), res, nil)
		}

		output.Success("Token rotat correctament per al túnel %s", res.TunnelID)
		fmt.Printf("Nou Tunnel Token: %s\n", res.TunnelToken)
		return nil
	},
}

var tunnelsDeleteCmd = &cobra.Command{
	Use:   "delete <tunnel-id>",
	Short: "Eliminar un túnel i totes les seves rutes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelID := args[0]
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		if !deleteTunnelForce {
			fmt.Printf("Esteu segur que voleu eliminar el túnel '%s' i TOTES les seves rutes? [s/N]: ", tunnelID)
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "s" && strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "si" {
				fmt.Println("Operació cancel·lada.")
				return nil
			}
		}

		if err := c.Tunnel.DeleteTunnel(cmd.Context(), tunnelID); err != nil {
			return err
		}

		output.Success("Túnel '%s' eliminat correctament.", tunnelID)
		return nil
	},
}

func init() {
	tunnelsListCmd.Flags().StringVar(&filterExternalID, "external-id", "", "Filtrar per un external_id concret")

	tunnelsCreateCmd.Flags().StringVar(&tunnelName, "name", "", "Nom o etiqueta del túnel")
	tunnelsCreateCmd.Flags().StringVar(&tunnelExternalID, "external-id", "", "Identificador extern únic propi per idempotència")
	tunnelsCreateCmd.Flags().StringVar(&tunnelSaveWGPath, "save-wg", "", "Ruta per desar el fitxer de configuració de WireGuard (ex. wg0.conf)")

	tunnelsDeleteCmd.Flags().BoolVarP(&deleteTunnelForce, "yes", "y", false, "Eliminar sense demanar confirmació interactiva")

	tunnelsCmd.AddCommand(tunnelsListCmd)
	tunnelsCmd.AddCommand(tunnelsGetCmd)
	tunnelsCmd.AddCommand(tunnelsCreateCmd)
	tunnelsCmd.AddCommand(tunnelsRotateTokenCmd)
	tunnelsCmd.AddCommand(tunnelsDeleteCmd)

	rootCmd.AddCommand(tunnelsCmd)
}
