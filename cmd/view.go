package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var outputJSON bool

var viewCmd = &cobra.Command{
	Use:   "view [contract-name]",
	Short: "Output detailed state and configuration for an environment",
	Long:  `Displays target settings, live metrics, and outputs for a specific infrastructure block.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contractName := args[0]
		
		if outputJSON {
			// Mocking JSON output for automation/scripts passing --json flag
			fmt.Printf(`{"contract": "%s", "status": "active", "outputs": {"alb_dns": "pay-alb-123.amazonaws.com"}}\n`, contractName)
			return
		}

		fmt.Printf("📄 Showing environment details for: %s\n", contractName)
		fmt.Println("======================================================")
		fmt.Println("Target Cloud:  AWS (us-east-1)")
		fmt.Println("Budget Ceiling: $450/month (Current Trend: $390)")
		fmt.Println("")
		fmt.Println("🔌 Live Infrastructure Outputs:")
		fmt.Println("  -> load_balancer_url: pay-alb-123.amazonaws.com")
		fmt.Println("  -> rds_endpoint:      pay-db.c123.us-east-1.rds.amazonaws.com")
	},
}

func init() {
	viewCmd.Flags().BoolVar(&outputJSON, "json", false, "Output runtime configurations in raw JSON format")
	rootCmd.AddCommand(viewCmd)
}