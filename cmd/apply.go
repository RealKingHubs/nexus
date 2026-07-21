package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nexus-io/nexus/pkg/backend"
	"github.com/nexus-io/nexus/pkg/engine"
	"github.com/nexus-io/nexus/pkg/provider"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	file        string
	autoApprove bool
	yesApprove  bool
)

var applyCmd = &cobra.Command{
	Use:     "apply [spec.yaml]",
	GroupID: "core",
	Short:   "Reconcile host or cloud infrastructure matching intent spec",
	Long:    `Applies an intent contract to reach target state. Automatically detects a YAML file in the current directory if no path is explicitly provided.`,
	Example: `  nexus apply
  nexus apply docker-test.yaml
  nexus apply aws-test.yaml -y`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Smart File Path Resolution
		if len(args) > 0 {
			file = args[0]
		}

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

		// 3. Output the speculative plan
		engine.PrintExecutionPlan(contract)

		// 4. Evaluate Confirmation Gate
		if autoApprove || yesApprove {
			fmt.Println("\n⚠️ Auto-approve flag detected. Bypassing interactive confirmation prompt...")
		} else {
			fmt.Print("\nDo you want to perform these actions?\n")
			fmt.Print("   Only 'yes' will be accepted to approve and proceed: ")

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

		// 5. Initialize Pluggable State Backend (Auto-discovers local file, nexus.yaml, or etcd)
		fmt.Println("\n🔌 Initializing state backend storage interface...")
		bk, err := backend.NewBackend()
		if err != nil {
			fmt.Printf("❌ Backend Initialization Error: %v\n", err)
			return
		}
		defer bk.Close()

		// 6. Claim the Distributed lease lock
		lockCtx, lockCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer lockCancel()

		fmt.Println("🔒 Requesting environment lease lock protection...")
		leaseID, acquired, err := bk.AcquireLock(lockCtx, "default-tenant", "production", contract.Metadata.Name, "cli-worker", 15)
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
			_ = bk.ReleaseLock(context.Background(), leaseID)
		}()

		fmt.Println("🟩 Lock Secured! Executing target infrastructure reconciliation...")

		// 7. Live Cloud Provider Interface Driver Processing Logic
		fmt.Printf("📡 Initializing live cloud provider infrastructure driver for: %s...\n", contract.Spec.Provider)
		var liveStatus engine.Status

		execCtx, execCancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer execCancel()

		switch contract.Spec.Provider {
		case "docker":
			prov, err := provider.NewDockerProvider()
			if err != nil {
				fmt.Printf("❌ Provider Setup Exception: %v\n", err)
				return
			}
			liveStatus, err = prov.Reconcile(execCtx, contract.Metadata.Name, contract.Spec)
			if err != nil {
				fmt.Printf("❌ Docker Orchestration Convergence Failure: %v\n", err)
				return
			}
		case "aws":
			prov, err := provider.NewAWSProvider(execCtx, contract.Spec.Region)
			if err != nil {
				fmt.Printf("❌ AWS Provider Setup Exception: %v\n", err)
				return
			}
			liveStatus, err = prov.Reconcile(execCtx, contract.Metadata.Name, contract.Spec)
			if err != nil {
				fmt.Printf("❌ AWS Infrastructure Convergence Failure: %v\n", err)
				return
			}
		default:
			fmt.Printf("❌ Provider '%s' is not supported. Supported providers: 'docker', 'aws'.\n", contract.Spec.Provider)
			return
		}

		contract.Status = liveStatus

		// 8. Serialize updated object containing spec + live status fields combined
		updatedBytes, err := yaml.Marshal(contract)
		if err != nil {
			fmt.Printf("❌ Failed to serialize runtime status updates: %v\n", err)
			return
		}

		// 9. Save to backend storage driver (Local file, S3, or Etcd)
		fmt.Println("💾 Saving updated intent state and outputs to state storage backend...")
		err = bk.PutContract(execCtx, "default-tenant", contract.Metadata.Name, updatedBytes)
		if err != nil {
			fmt.Printf("❌ Storage Write Error: %v\n", err)
			return
		}

		// 10. Display clean runtime outputs matrix directly to user terminal
		fmt.Println("\n📋 Orchestration Outputs:")
		fmt.Println("----------------------------------------------------------------------")
		for key, val := range contract.Status.Outputs {
			fmt.Printf("🔹 %-15s = %s\n", key, val)
		}
		fmt.Println("----------------------------------------------------------------------")
		fmt.Println("✨ Infrastructure state matches intent perfectly. Run finalized successfully!")
	},
}

func init() {
	applyCmd.Flags().StringVarP(&file, "file", "f", "", "Explicit path to the intent contract YAML file")
	applyCmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive confirmation and deploy immediately")
	applyCmd.Flags().BoolVarP(&yesApprove, "yes", "y", false, "Skip interactive confirmation and deploy immediately")

	rootCmd.AddCommand(applyCmd)
}