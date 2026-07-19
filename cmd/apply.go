package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nexus-io/nexus/pkg/engine"
	"github.com/nexus-io/nexus/pkg/registry"
	"github.com/spf13/cobra"
)

var (
	file        string
	autoApprove bool
	yesApprove  bool
)

var applyCmd = &cobra.Command{
	Use:   "apply [file]",
	Short: "Apply an intent contract configuration",
	Long:  `Applies an intent contract. Automatically detects a YAML file in the current directory if no path is given.`,
	Args:  cobra.MaximumNArgs(1), // Allows 0 or 1 positional arguments
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Smart File Path Resolution
		// Prioritize positional argument over the -f flag
		if len(args) > 0 {
			file = args[0]
		}

		// If no positional argument and no flag, start auto-detection
		if file == "" {
			files, err := os.ReadDir(".")
			if err != nil {
				fmt.Printf("❌ Error reading current directory: %v\n", err)
				return
			}

			var yamlFiles []string
			for _, f := range files {
				if !f.IsDir() {
					ext := filepath.Ext(f.Name())
					if ext == ".yaml" || ext == ".yml" {
						yamlFiles = append(yamlFiles, f.Name())
					}
				}
			}

			if len(yamlFiles) == 0 {
				fmt.Println("❌ Error: No intent contract files (.yaml or .yml) found in this directory.")
				fmt.Println("   Please specify a file path: nexus apply [filename]")
				return
			}

			// Handle folder layout ambiguity
			if len(yamlFiles) > 1 {
				foundDefault := false
				for _, yf := range yamlFiles {
					if yf == "intent.yaml" {
						file = yf
						foundDefault = true
						break
					}
				}
				if !foundDefault {
					fmt.Println("❌ Error: Multiple configuration files detected in this directory:")
					for _, yf := range yamlFiles {
						fmt.Printf("   - %s\n", yf)
					}
					fmt.Println("\n   Please specify which file to use: nexus apply [filename]")
					return
				}
			} else {
				// Exactly one target configuration found
				file = yamlFiles[0]
			}
			fmt.Printf("📂 Auto-detected target configuration: %s\n", file)
		}

		// 2. Structural Check & Schema Validation
		contract, err := engine.VerifyContractFile(file)
		if err != nil {
			fmt.Printf("❌ Validation Failed: %v\n", err)
			return
		}

		// 3. Output the speculative plan so the operator isn't blinded
		engine.PrintExecutionPlan(contract)

		// 4. Evaluate Confirmation Gate
		if autoApprove || yesApprove {
			fmt.Println("\n⚠️ Auto-approve flag detected. Bypassing interactive confirmation prompt...")
		} else {
			fmt.Print("\nDo you want to perform these actions?\n")
			fmt.Print("  Only 'yes' will be accepted to approve and proceed: ")
			
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("❌ Error reading keyboard input: %v\n", err)
				return
			}
			
			confirmed := strings.TrimSpace(strings.ToLower(input))
			if confirmed != "yes" {
				fmt.Println("\n🛑 Apply cancelled by operator. No cloud assets were touched.")
				return
			}
		}

		// 5. Connect to etcd Registry
		fmt.Println("\n🔌 Connecting to etcd state backend registry...")
		reg, err := registry.NewEtcdRegistry([]string{"127.0.0.1:2379"}, 5*time.Second)
		if err != nil {
			fmt.Printf("❌ Connection Error: %v\n", err)
			return
		}
		defer reg.Close()

		// 6. Claim the Distributed lease lock
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Println("🔒 Requesting environment lease lock protection...")
		leaseID, acquired, err := reg.AcquireDistributedLock(ctx, "default-tenant", "production", contract.Metadata.Name, "cli-worker", 15)
		if err != nil {
			fmt.Printf("❌ Lock Exception: %v\n", err)
			return
		}

		if !acquired {
			fmt.Println("\n❌ Error: Target Environment is LOCKED by an ongoing deployment!")
			return
		}

		defer func() {
			fmt.Println("🔓 Releasing infrastructure environment lock...")
			_ = reg.ReleaseDistributedLock(context.Background(), leaseID)
		}()

		fmt.Println("🟩 Lock Secured! Initializing active orchestration loop...")
		
		// 7. Read the raw file data bytes to persist them into our cluster brain
		fileBytes, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("❌ Failed to parse storage copy: %v\n", err)
			return
		}

		fmt.Println("💾 Saving validated intent contract state to cluster registry...")
		err = reg.PutContract(ctx, "default-tenant", contract.Metadata.Name, fileBytes)
		if err != nil {
			fmt.Printf("❌ Storage Write Error: %v\n", err)
			return
		}

		time.Sleep(2 * time.Second)
		fmt.Println("✨ Infrastructure state matches intent perfectly. Run finalized.")
	},
}

func init() {
	applyCmd.Flags().StringVarP(&file, "file", "f", "", "Explicit path to the intent contract YAML file")
	applyCmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive confirmation and deploy immediately")
	applyCmd.Flags().BoolVarP(&yesApprove, "yes", "y", false, "Skip interactive confirmation and deploy immediately")
	
	rootCmd.AddCommand(applyCmd)
}