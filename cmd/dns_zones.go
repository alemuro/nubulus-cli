package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"nubulusttun/pkg/output"
)

var deleteZoneForce bool

var zonesCmd = &cobra.Command{
	Use:     "zones",
	Aliases: []string{"zone", "z"},
	Short:   "Administrar zones DNS",
	Long:    "Ordres per a llistar, crear, obtenir detalls, verificar i eliminar zones DNS.",
}

var zonesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Llistar totes les zones DNS del compte",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		zones, err := c.DNS.ListZones(cmd.Context())
		if err != nil {
			return err
		}

		return output.PrintStructured(os.Stdout, output.Format(outputFormat), zones, func(w io.Writer) error {
			if len(zones) == 0 {
				fmt.Fprintln(w, "No s'ha trobat cap zona DNS.")
				return nil
			}

			tw := output.NewTableWriter(w)
			fmt.Fprintln(tw, "ID\tNOM DE ZONA\tORIGEN\tESTAT\tVERIFICAT\tCREAT")
			for _, z := range zones {
				verified := "-"
				if z.VerifiedAt != nil {
					verified = z.VerifiedAt.Format(time.RFC3339)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
					z.ID,
					z.Name,
					z.Source,
					z.Status,
					verified,
					z.CreatedAt.Format("2006-01-02 15:04:05"),
				)
			}
			return tw.Flush()
		})
	},
}

var zonesGetCmd = &cobra.Command{
	Use:   "get <nom-zona>",
	Short: "Obtenir informació detallada d'una zona DNS",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		zoneName := args[0]
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		detail, err := c.DNS.GetZone(cmd.Context(), zoneName)
		if err != nil {
			return err
		}

		return output.PrintStructured(os.Stdout, output.Format(outputFormat), detail, func(w io.Writer) error {
			tw := output.NewTableWriter(w)
			fmt.Fprintf(tw, "ID:\t%s\n", detail.Zone.ID)
			fmt.Fprintf(tw, "Nom:\t%s\n", detail.Zone.Name)
			fmt.Fprintf(tw, "Estat:\t%s\n", detail.Zone.Status)
			fmt.Fprintf(tw, "Origen:\t%s\n", detail.Zone.Source)
			fmt.Fprintf(tw, "Account ID:\t%s\n", detail.Zone.AccountID)
			if detail.Serial != nil {
				fmt.Fprintf(tw, "Serial SOA:\t%d\n", *detail.Serial)
			}
			if len(detail.Nameservers) > 0 {
				fmt.Fprintf(tw, "Nameservers:\t%s\n", strings.Join(detail.Nameservers, ", "))
			}
			if detail.PrimaryError != "" {
				fmt.Fprintf(tw, "Error Primari:\t%s\n", detail.PrimaryError)
			}
			if detail.Zone.VerifiedAt != nil {
				fmt.Fprintf(tw, "Verificat:\t%s\n", detail.Zone.VerifiedAt.Format(time.RFC3339))
			}
			if detail.Zone.ReservedUntil != nil {
				fmt.Fprintf(tw, "Reserva fins:\t%s\n", detail.Zone.ReservedUntil.Format(time.RFC3339))
			}
			fmt.Fprintf(tw, "Creat:\t%s\n", detail.Zone.CreatedAt.Format("2006-01-02 15:04:05"))

			if detail.Verification != nil && detail.Verification.Required {
				fmt.Fprintln(tw, "\n--- Dades de Verificació ---")
				fmt.Fprintf(tw, "Host TXT:\t%s\n", detail.Verification.TXTRecordHost)
				fmt.Fprintf(tw, "Valor TXT:\t%s\n", detail.Verification.TXTRecordValue)
				if len(detail.Verification.Nameservers) > 0 {
					fmt.Fprintf(tw, "NS Objectiu:\t%s\n", strings.Join(detail.Verification.Nameservers, ", "))
				}
			}
			return tw.Flush()
		})
	},
}

var zonesCreateCmd = &cobra.Command{
	Use:   "create <nom-zona>",
	Short: "Crear o reclamar una nova zona DNS",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		zoneName := args[0]
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		detail, err := c.DNS.CreateZone(cmd.Context(), zoneName)
		if err != nil {
			return err
		}

		if outputFormat == "json" || outputFormat == "yaml" {
			return output.PrintStructured(os.Stdout, output.Format(outputFormat), detail, nil)
		}

		output.Success("Zona '%s' registrada amb èxit (ID: %s, Estat: %s)", detail.Zone.Name, detail.Zone.ID, detail.Zone.Status)
		if detail.Verification != nil && detail.Verification.Required {
			output.Warn("La zona és externa i requereix verificació de propietat:")
			fmt.Printf("  • Creeu un registre TXT a: %s\n", detail.Verification.TXTRecordHost)
			fmt.Printf("  • Amb el valor: %s\n", detail.Verification.TXTRecordValue)
			fmt.Printf("  • Delegueu als servidors de noms: %s\n", strings.Join(detail.Verification.Nameservers, ", "))
			fmt.Printf("\nUn cop publicat, executeu: nubulus zones verify %s\n", detail.Zone.Name)
		}
		return nil
	},
}

var zonesDeleteCmd = &cobra.Command{
	Use:   "delete <nom-zona>",
	Short: "Eliminar una zona DNS de forma permanent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		zoneName := args[0]
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		if !deleteZoneForce {
			fmt.Printf("Esteu segur que voleu eliminar la zona DNS '%s'? [s/N]: ", zoneName)
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "s" && strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "si" {
				fmt.Println("Operació cancel·lada.")
				return nil
			}
		}

		if err := c.DNS.DeleteZone(cmd.Context(), zoneName); err != nil {
			return err
		}

		output.Success("Zona DNS '%s' eliminada correctament.", zoneName)
		return nil
	},
}

var zonesVerificationCmd = &cobra.Command{
	Use:     "verification <nom-zona>",
	Aliases: []string{"challenge"},
	Short:   "Consultar les instruccions del repte de verificació d'una zona",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		zoneName := args[0]
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		verif, err := c.DNS.Verification(cmd.Context(), zoneName)
		if err != nil {
			return err
		}

		return output.PrintStructured(os.Stdout, output.Format(outputFormat), verif, func(w io.Writer) error {
			tw := output.NewTableWriter(w)
			fmt.Fprintf(tw, "Zona:\t%s\n", verif.Zone)
			fmt.Fprintf(tw, "Estat:\t%s\n", verif.Status)
			fmt.Fprintf(tw, "Verificació Requerida:\t%t\n", verif.Required)
			fmt.Fprintf(tw, "Host TXT:\t%s\n", verif.TXTRecordHost)
			fmt.Fprintf(tw, "Valor TXT:\t%s\n", verif.TXTRecordValue)
			if len(verif.Nameservers) > 0 {
				fmt.Fprintf(tw, "Nameservers:\t%s\n", strings.Join(verif.Nameservers, ", "))
			}
			if verif.ReservedUntil != nil {
				fmt.Fprintf(tw, "Reserva activa fins:\t%s\n", verif.ReservedUntil.Format(time.RFC3339))
			}
			if verif.VerifiedAt != nil {
				fmt.Fprintf(tw, "Verificat el:\t%s\n", verif.VerifiedAt.Format(time.RFC3339))
			}
			return tw.Flush()
		})
	},
}

var zonesVerifyCmd = &cobra.Command{
	Use:   "verify <nom-zona>",
	Short: "Executar una comprovació activa de verificació de propietat",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		zoneName := args[0]
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		res, err := c.DNS.Verify(cmd.Context(), zoneName)
		if err != nil {
			return err
		}

		if outputFormat == "json" || outputFormat == "yaml" {
			return output.PrintStructured(os.Stdout, output.Format(outputFormat), res, nil)
		}

		if res.Verified {
			output.Success("Zona '%s' verificada amb èxit mitjançant mètode '%s'!", res.Zone, res.Method)
		} else {
			output.Warn("La verificació de la zona '%s' encara no ha tingut èxit.", res.Zone)
			fmt.Printf("  • Codi de motiu: %s\n", res.ReasonCode)
			fmt.Printf("  • Motiu: %s\n", res.Reason)
			fmt.Println("\nNota: Si heu creat el registre TXT recentment, tingueu en compte el temps de propagació/TTL DNS.")
		}
		return nil
	},
}

func init() {
	zonesDeleteCmd.Flags().BoolVarP(&deleteZoneForce, "yes", "y", false, "Eliminar sense demanar confirmació interactiva")

	zonesCmd.AddCommand(zonesListCmd)
	zonesCmd.AddCommand(zonesGetCmd)
	zonesCmd.AddCommand(zonesCreateCmd)
	zonesCmd.AddCommand(zonesDeleteCmd)
	zonesCmd.AddCommand(zonesVerificationCmd)
	zonesCmd.AddCommand(zonesVerifyCmd)

	rootCmd.AddCommand(zonesCmd)
}
