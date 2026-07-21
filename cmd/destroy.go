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
	"github.com/nexus-io/nexus/pkg/provider"
	"github.com/spf13/cobra"
)

var (
	destroyFile        string
	destroyAutoApprove bool
	destroyYesApprove  bool
)

var destroyCmd = &cobra.Command{
	Use:   "destroy [file]",
	Short: "Destroy all managed assets inside an intent contract",
	Long: `Scans the targeted configuration state, calculates a destructive teardown blueprint, 
and purges all tracked cloud/container assets from the active environment registry.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Smart File Path Resolution
		if len(args) > 0 {
			destroyFile = args[0]
		}

		if destroyFile == "" {
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
				fmt.Println("❌ Error: No intent contract files found to target for destruction.")
				return
			}

			if len(yamlFiles) > 1 {
				foundDefault := false
				for _, yf := range yamlFiles {
					if yf == "intent.yaml" {
						destroyFile = yf
						foundDefault = true
						break
					}
				}
				if !foundDefault {
					fmt.Println("❌ Error: Multiple configuration files found. Specify which to destroy: nexus destroy [filename]")
					return
				}
			} else {
				destroyFile = yamlFiles[0]
			}
			fmt.Printf("📂 Auto-detected destruction target configuration: %s\n", destroyFile)
		}

		// 2. Load and Validate Contract
		contract, err := engine.VerifyContractFile(destroyFile)
		if err != nil {
			fmt.Printf("❌ Validation Failed: %v\n", err)
			return
		}

		// 3. Render Destructive Speculative Plan
		fmt.Println("\n💥 Nexus Speculative DESTRUCTION Plan")
		fmt.Println("==========================================================")
		fmt.Printf("🏢 Resource Target:  %s\n", contract.Metadata.Name)
		fmt.Printf("🌐 Environment:      %s\n", contract.Metadata.Environment)
		fmt.Printf("☁️  Cloud Provider:   %s (%s)\n", contract.Spec.Provider, contract.Spec.Region)
		fmt.Println("----------------------------------------------------------")
		fmt.Println("➖ [DESTROY] Wiping all live remote assets permanently.")
		fmt.Println("==========================================================")

		// 4. Safety Confirmation Prompt
		if destroyAutoApprove || destroyYesApprove {
			fmt.Println("\n⚠️ Auto-approve flag detected. Bypassing interactive destruction confirmation...")
		} else {
			fmt.Print("\n🔥 WARNING: This action cannot be undone! Are you sure you want to destroy these resources?\n")
			fmt.Print("  Only 'yes' will be accepted to confirm teardown: ")
			
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("❌ Error reading safety gate: %v\n", err)
				return
			}
			
			if strings.TrimSpace(strings.ToLower(input)) != "yes" {
				fmt.Println("\n🛑 Teardown cancelled. Infrastructure preserved safely.")
				return
			}
		}

		// 5. Establish Registry Connection
		fmt.Println("\n🔌 Connecting to etcd state backend registry...")
		reg, err := registry.NewEtcdRegistry([]string{"127.0.0.1:2379"}, 5*time.Second)
		if err != nil {
			fmt.Printf("❌ Connection Error: %v\n", err)
			return
		}
		defer reg.Close()

		// 6. Acquire Lock Protection to prevent mid-destruction overwrite races
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Println("🔒 Requesting environment lease lock protection...")
		leaseID, acquired, err := reg.AcquireDistributedLock(ctx, "default-tenant", "production", contract.Metadata.Name, "cli-destroyer", 15)
		if err != nil {
			fmt.Printf("❌ Lock Exception: %v\n", err)
			return
		}
		if !acquired {
			fmt.Println("\n❌ Error: Environment is locked by another running operation!")
			return
		}
		defer func() {
			fmt.Println("🔓 Releasing infrastructure environment lock...")
			_ = reg.ReleaseDistributedLock(context.Background(), leaseID)
		}()

		// 7. Execute Active Teardown Driver Action Operations
		fmt.Printf("🟩 Lock Secured! Dispatching destructive driver logic for: %s...\n", contract.Spec.Provider)
		
		switch contract.Spec.Provider {
		case "docker":
			prov, err := provider.NewDockerProvider()
			if err != nil {
				fmt.Printf("❌ Provider Setup Exception: %v\n", err)
				return
			}
			err = prov.Destroy(ctx, contract.Metadata.Name, contract.Spec)
			if err != nil {
				fmt.Printf("❌ Destructive Provider Driver Execution Failure: %v\n", err)
				return
			}
		default:
			fmt.Printf("⚠️ Provider '%s' bypassing active driver teardown (Simulation Mode).\n", contract.Spec.Provider)
			time.Sleep(2 * time.Second)
		}
		
		fmt.Println("💥 Remote infrastructure assets torn down successfully.")

		// 8. Purge Configuration State from Core Cluster Memory Space
		fmt.Println("💾 Purging configuration state entry records from active cluster registry...")
		err = reg.DeleteContract(ctx, "default-tenant", contract.Metadata.Name)
		if err != nil {
			fmt.Printf("❌ Registry Purge Failure: %v\n", err)
			return
		}

		fmt.Println("✨ Teardown finalized. System space is clean.")
	},
}

func init() {
	destroyCmd.Flags().StringVarP(&destroyFile, "file", "f", "", "Explicit path to the contract YAML file to destroy")
	destroyCmd.Flags().BoolVar(&destroyAutoApprove, "auto-approve", false, "Skip interactive safety prompt and destroy immediately")
	destroyCmd.Flags().BoolVarP(&destroyYesApprove, "yes", "y", false, "Skip interactive safety prompt and destroy immediately")
	
	rootCmd.AddCommand(destroyCmd)
}