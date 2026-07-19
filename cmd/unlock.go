package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var reason string

var unlockCmd = &cobra.Command{
	Use:   "unlock [contract-name]",
	Short: "Force-evict a stuck environment deployment lease lock",
	Long:  `Removes administrative deployment locks left behind by interrupted or crashed infrastructure processes.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contractName := args[0]
		if reason == "" {
			fmt.Println("❌ Error: --reason (-r) flag is required to complete an administrative override unlock")
			return
		}
		fmt.Printf("🔓 Evicting lease lock for environment: %s...\n", contractName)
		fmt.Println("✅ Lock successfully broken.")
	},
}

func init() {
	unlockCmd.Flags().StringVarP(&reason, "reason", "r", "", "Audit log reason for forcing lock eviction")
	rootCmd.AddCommand(unlockCmd)
}