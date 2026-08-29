package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/alemuro/nubulus-cli/pkg/output"
)

//go:embed skills/SKILL.md
var skillContent string

var (
	agentTargetDir string
	agentForce     bool
)

var agentsCmd = &cobra.Command{
	Use:     "agents",
	Aliases: []string{"agent", "skills", "skill"},
	Short:   "Administrar skills per a agents d'intel·ligència artificial (Antigravity/Claude)",
	Long:    "Ordres per a instal·lar i gestionar les skills de Nubulus perquè els agents de codificació sàpiguen utilitzar la CLI.",
}

var agentsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Instal·lar la skill de Nubulus al directori global d'agents",
	Long: `Copia la definició de la skill (SKILL.md) als directoris globals d'agents
com ~/.gemini/config/skills/nubulus/SKILL.md i ~/.config/antigravity/skills/nubulus/SKILL.md.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetDirs []string

		if agentTargetDir != "" {
			targetDirs = append(targetDirs, agentTargetDir)
		} else {
			home := os.Getenv("HOME")
			candidates := []string{
				filepath.Join(home, ".gemini", "config", "skills"),
				filepath.Join(home, ".config", "antigravity", "skills"),
				filepath.Join(home, ".claude", "skills"),
			}

			for _, c := range candidates {
				// Check if parent directory exists or create if standard .gemini exists
				parent := filepath.Dir(c)
				if _, err := os.Stat(parent); err == nil {
					targetDirs = append(targetDirs, c)
				}
			}

			// If none found, default to standard ~/.gemini/config/skills
			if len(targetDirs) == 0 {
				targetDirs = append(targetDirs, filepath.Join(home, ".gemini", "config", "skills"))
			}
		}

		installedCount := 0
		for _, dir := range targetDirs {
			skillDir := filepath.Join(dir, "nubulus")
			if err := os.MkdirAll(skillDir, 0755); err != nil {
				output.Warn("No s'ha pogut crear el directori '%s': %v", skillDir, err)
				continue
			}

			skillFilePath := filepath.Join(skillDir, "SKILL.md")
			if err := os.WriteFile(skillFilePath, []byte(skillContent), 0644); err != nil {
				output.Warn("No s'ha pogut escriure '%s': %v", skillFilePath, err)
				continue
			}

			output.Success("Skill 'nubulus' instal·lada amb èxit a: %s", skillFilePath)
			installedCount++
		}

		if installedCount == 0 {
			return fmt.Errorf("no s'ha pogut instal·lar la skill en cap directori")
		}

		fmt.Println("\nEls agents d'IA ara reconeixeran automàticament les capacitats de 'nubulus' (exposició de ports, DNS, túnels i rutes).")
		return nil
	},
}

var agentsShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Mostrar el contingut de la skill incrustada",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(skillContent)
	},
}

func init() {
	agentsInstallCmd.Flags().StringVar(&agentTargetDir, "dir", "", "Directori destí personalitzat per a la skill")
	agentsInstallCmd.Flags().BoolVarP(&agentForce, "force", "f", false, "Forçar la sobreescriptura de la skill")

	agentsCmd.AddCommand(agentsInstallCmd)
	agentsCmd.AddCommand(agentsShowCmd)

	rootCmd.AddCommand(agentsCmd)
}
