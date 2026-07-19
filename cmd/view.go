package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/nexus-io/nexus/pkg/registry"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view [contract-name]",
	Short: "View details of a specific managed intent contract",
	Long:  `Queries the live etcd cluster brain and displays the raw, comment-preserved YAML configuration state for the specified target.`,
	Args:  cobra.ExactArgs(1), // Enforces that exactly 1 positional argument must be passed
	Run: func(cmd *cobra.Command, args []string) {
		contractName := args[0]

		fmt.Printf("🔌 Fetching intent contract '%s' from Nexus registry...\n", contractName)

		// 1. Connect to the state backend cluster
		reg, err := registry.NewEtcdRegistry([]string{"127.0.0.1:2379"}, 5*time.Second)
		if err != nil {
			fmt.Printf("❌ Registry Connection Error: %v\n", err)
			return
		}
		defer reg.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 2. Query the exact contract key path
		rawBytes, err := reg.GetContract(ctx, "default-tenant", contractName)
		if err != nil {
			fmt.Printf("❌ Read Operation Failure: %v\n", err)
			return
		}

		// 3. Handle data absences gracefully
		if rawBytes == nil {
			fmt.Printf("\n🍁 Error: No active intent contract named '%s' exists in this environment space.\n", contractName)
			fmt.Println("   -> Run './bin/nexus list' to view all available active contracts.")
			return
		}

		// 4. Output the raw configuration text dynamically
		fmt.Println("")
		fmt.Printf("📄 ACTIVE CONFIGURATION STATE FOR: %s\n", contractName)
		fmt.Println("======================================================================")
		fmt.Println(string(rawBytes))
		fmt.Println("======================================================================")
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
}