package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version = "1.0.0"
	Commit  = "none"
	Date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Mostra la versió de la CLI de Nubulus",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("nubulus CLI v%s (commit: %s, data: %s)\n", Version, Commit, Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
