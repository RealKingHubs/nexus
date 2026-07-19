package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nexus",
	Short: "Nexus is an intent-driven cloud-native infrastructure control plane",
	Long: `Nexus acts as a declarative coordination layer sitting over cloud resources. 
It processes intent contracts, locks cluster states via distributed leases, 
validates schemas, and enforces real-world cloud infrastructure convergence.

Complete documentation available at: https://github.com/nexus-io/nexus`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ CLI Execution Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Root flags can be declared here if needed across all commands globally
}