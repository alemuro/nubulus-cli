package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"nubulusttun/pkg/client"
	"nubulusttun/pkg/output"
)

var (
	recordTTL         uint32
	deleteRecordForce bool
)

var recordsCmd = &cobra.Command{
	Use:     "records",
	Aliases: []string{"record", "rrset", "rrsets", "dns"},
	Short:   "Administrar registres DNS (RRsets)",
	Long:    "Ordres per a llistar, consultar, afegir/actualitzar i eliminar conjunts de registres DNS (RRsets).",
}

var recordsListCmd = &cobra.Command{
	Use:     "list <nom-zona>",
	Aliases: []string{"ls"},
	Short:   "Llistar tots els registres DNS d'una zona",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		zoneName := args[0]
		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		content, err := c.DNS.ZoneContent(cmd.Context(), zoneName)
		if err != nil {
			return err
		}

		return output.PrintStructured(os.Stdout, output.Format(outputFormat), content, func(w io.Writer) error {
			if len(content.RRsets) == 0 {
				fmt.Fprintf(w, "La zona '%s' no conté registres o encara no s'ha pogut llegir del primari.\n", zoneName)
				return nil
			}

			tw := output.NewTableWriter(w)
			fmt.Fprintln(tw, "NOM (FQDN)\tTIPUS\tTTL\tVALORS")
			for _, r := range content.RRsets {
				valStr := strings.Join(r.Values, ", ")
				if len(valStr) > 60 {
					valStr = valStr[:57] + "..."
				}
				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n",
					r.Name,
					r.Type,
					r.TTL,
					valStr,
				)
			}
			return tw.Flush()
		})
	},
}

var recordsGetCmd = &cobra.Command{
	Use:   "get <nom-zona> <nom-registre> <tipus>",
	Short: "Consultar un registre DNS concret",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		zoneName := args[0]
		recordName := args[1]
		recordType := args[2]

		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		fqdn, ok := client.QualifyName(recordName, zoneName)
		if !ok {
			return fmt.Errorf("el nom de registre '%s' no és vàlid per a la zona '%s'", recordName, zoneName)
		}

		rrset, err := c.DNS.GetRRset(cmd.Context(), zoneName, fqdn, recordType)
		if err != nil {
			return err
		}

		return output.PrintStructured(os.Stdout, output.Format(outputFormat), rrset, func(w io.Writer) error {
			tw := output.NewTableWriter(w)
			fmt.Fprintf(tw, "Nom FQDN:\t%s\n", rrset.Name)
			fmt.Fprintf(tw, "Tipus:\t%s\n", rrset.Type)
			fmt.Fprintf(tw, "TTL:\t%d segons\n", rrset.TTL)
			fmt.Fprintln(tw, "Valors:")
			for _, v := range rrset.Values {
				fmt.Fprintf(tw, "  • %s\n", v)
			}
			return tw.Flush()
		})
	},
}

var recordsSetCmd = &cobra.Command{
	Use:   "set <nom-zona> <nom-registre> <tipus> <valor...>",
	Short: "Crear o actualitzar un conjunt de registres DNS (RRset)",
	Long: `Crea o sobreescriu completament els valors d'un RRset.

Exemples:
  # Crear un registre A
  nubulus records set example.com www A 198.51.100.10 --ttl 300

  # Crear múltiples registres A (round-robin)
  nubulus records set example.com api A 198.51.100.10 198.51.100.11 --ttl 300

  # Crear un registre CNAME apuntant a un túnel
  nubulus records set example.com app CNAME tun-xyz.nubulustun.com.

  # Crear un registre MX a l'apex (@)
  nubulus records set example.com @ MX "10 mail.example.com."
`,
	Args: cobra.MinimumNArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		zoneName := args[0]
		recordName := args[1]
		recordType := strings.ToUpper(args[2])
		values := args[3:]

		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		fqdn, ok := client.QualifyName(recordName, zoneName)
		if !ok {
			return fmt.Errorf("el nom '%s' no es pot ubicar dins la zona '%s'", recordName, zoneName)
		}

		// Validations
		if _, forbidden := client.ForbiddenTypes[recordType]; forbidden {
			return fmt.Errorf("el tipus de registre '%s' no està permès per l'API", recordType)
		}
		if _, managed := client.ManagedAtApex[recordType]; managed && (recordType == "SOA" || client.IsApex(fqdn, zoneName)) {
			return fmt.Errorf("el registre %s a l'apex de la zona és gestionat per la plataforma i no es pot editar", recordType)
		}
		if recordType == "CNAME" {
			if client.IsApex(fqdn, zoneName) {
				return fmt.Errorf("un registre CNAME no pot estar a l'apex de la zona")
			}
			if len(values) != 1 {
				return fmt.Errorf("un registre CNAME ha de tenir exactament 1 valor")
			}
		}

		if recordTTL < client.MinTTL || recordTTL > client.MaxTTL {
			return fmt.Errorf("el TTL ha d'estar entre %d i %d segons", client.MinTTL, client.MaxTTL)
		}

		out, err := c.DNS.PutRRset(cmd.Context(), zoneName, fqdn, recordType, recordTTL, values)
		if err != nil {
			return err
		}

		if outputFormat == "json" || outputFormat == "yaml" {
			return output.PrintStructured(os.Stdout, output.Format(outputFormat), out, nil)
		}

		output.Success("Registre %s a '%s' desat correctament (TTL: %d, %d valors)", out.Type, out.Name, out.TTL, len(out.Values))
		return nil
	},
}

var recordsDeleteCmd = &cobra.Command{
	Use:   "delete <nom-zona> <nom-registre> <tipus>",
	Short: "Eliminar un conjunt de registres DNS",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		zoneName := args[0]
		recordName := args[1]
		recordType := strings.ToUpper(args[2])

		c, err := getClient(cmd.Context())
		if err != nil {
			return err
		}

		fqdn, ok := client.QualifyName(recordName, zoneName)
		if !ok {
			return fmt.Errorf("nom '%s' no vàlid per a la zona '%s'", recordName, zoneName)
		}

		if !deleteRecordForce {
			fmt.Printf("Esteu segur que voleu eliminar el registre %s a '%s'? [s/N]: ", recordType, fqdn)
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "s" && strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "si" {
				fmt.Println("Operació cancel·lada.")
				return nil
			}
		}

		if err := c.DNS.DeleteRRset(cmd.Context(), zoneName, fqdn, recordType); err != nil {
			return err
		}

		output.Success("Registre %s a '%s' eliminat correctament.", recordType, fqdn)
		return nil
	},
}

func init() {
	recordsSetCmd.Flags().Uint32Var(&recordTTL, "ttl", 300, "TTL en segons (60 - 604800)")
	recordsDeleteCmd.Flags().BoolVarP(&deleteRecordForce, "yes", "y", false, "Eliminar sense demanar confirmació interactiva")

	recordsCmd.AddCommand(recordsListCmd)
	recordsCmd.AddCommand(recordsGetCmd)
	recordsCmd.AddCommand(recordsSetCmd)
	recordsCmd.AddCommand(recordsDeleteCmd)

	rootCmd.AddCommand(recordsCmd)
}
