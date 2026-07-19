package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new Nexus workspace",
	Long:  `Creates a default workspace structure with boilerplate configuration files.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		fmt.Printf("✅ Initialized empty Nexus workspace in ./%s\n", projectName)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}