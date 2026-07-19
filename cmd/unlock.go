package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nexus-io/nexus/pkg/registry"
	"github.com/spf13/cobra"
)

var (
	unlockEnv         string
	unlockForceBypass bool
)

var unlockCmd = &cobra.Command{
	Use:   "unlock [contract-name]",
	Short: "Forcefully release a stalled environment lock",
	Long:  `Deletes an active distributed lock from the etcd registry to recover from crashed or hung deployment pipelines.`,
	Args:  cobra.ExactArgs(1), // Requires exactly the name of the locked contract
	Run: func(cmd *cobra.Command, args []string) {
		contractName := args[0]

		// 1. Safety Gate Prompt
		if !unlockForceBypass {
			fmt.Printf("⚠️ WARNING: Forcefully unlocking '%s' in the '%s' environment can cause data corruption or split-brain deployments if another worker process is still running.\n", contractName, unlockEnv)
			fmt.Print("Are you absolutely sure you want to proceed? (Type 'yes' to confirm): ")
			
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("❌ Error reading safety verification: %v\n", err)
				return
			}
			
			if strings.TrimSpace(strings.ToLower(input)) != "yes" {
				fmt.Println("🛑 Administrative unlock aborted by operator.")
				return
			}
		}

		fmt.Printf("🔓 Breaking active distributed lock lease for target: %s...\n", contractName)

		// 2. Establish connection to the state engine
		reg, err := registry.NewEtcdRegistry([]string{"127.0.0.1:2379"}, 5*time.Second)
		if err != nil {
			fmt.Printf("❌ Registry Connection Error: %v\n", err)
			return
		}
		defer reg.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 3. Purge the lock tracking path out of the storage cluster
		err = reg.ForceUnlock(ctx, "default-tenant", unlockEnv, contractName)
		if err != nil {
			fmt.Printf("❌ Administrative Operation Failed: %v\n", err)
			return
		}

		fmt.Println("✨ Environment lock cleared successfully. The orchestration pipeline is now free.")
	},
}

func init() {
	unlockCmd.Flags().StringVarP(&unlockEnv, "environment", "e", "production", "Target environment for the lock override")
	unlockCmd.Flags().BoolVarP(&unlockForceBypass, "yes", "y", false, "Bypass interactive confirmation safety prompt")
	rootCmd.AddCommand(unlockCmd)
}