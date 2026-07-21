package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nexus-io/nexus/pkg/engine"
	"github.com/spf13/cobra"
)

var verifyFile string

var verifyCmd = &cobra.Command{
	Use:     "verify [spec.yaml]",
	GroupID: "core",
	Short:   "Verify contract syntax and policies without applying changes",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			verifyFile = args[0]
		}

		if verifyFile == "" {
			// Auto-detect single yaml or intent.yaml
			files, err := os.ReadDir(".")
			if err != nil {
				fmt.Printf("❌ Error reading current directory: %v\n", err)
				return
			}
			for _, f := range files {
				if !f.IsDir() && (filepath.Ext(f.Name()) == ".yaml" || filepath.Ext(f.Name()) == ".yml") {
					verifyFile = f.Name()
					break
				}
			}
		}

		if verifyFile == "" {
			fmt.Println("❌ Error: No intent contract file found. Specify one with: nexus verify [filename]")
			return
		}

		fmt.Printf("🔍 Nexus Engine evaluating file target: %s...\n", verifyFile)
		contract, err := engine.VerifyContractFile(verifyFile)
		if err != nil {
			fmt.Printf("❌ Validation Failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("🟩 Contract syntax and policy validation passed successfully!")
		engine.PrintExecutionPlan(contract)
	},
}

func init() {
	verifyCmd.Flags().StringVarP(&verifyFile, "file", "f", "", "Explicit path to the intent contract YAML file")
	rootCmd.AddCommand(verifyCmd)
}