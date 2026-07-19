package cmd

import (
	"fmt"
	"github.com/nexus-io/nexus/pkg/engine" // Importing your backend engine logic
	"github.com/spf13/cobra"
)

var verifyFile string

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify contract syntax and policies without applying changes",
	Run: func(cmd *cobra.Command, args []string) {
		if verifyFile == "" {
			fmt.Println("❌ Error: Path to contract file required via -f or --file")
			return
		}
		
		fmt.Printf("🔍 Nexus Engine evaluating file target: %s...\n", verifyFile)
		
		// 🔌 Fixed: Added the blank identifier '_' to catch the contract return value
		_, err := engine.VerifyContractFile(verifyFile)
		if err != nil {
			fmt.Printf("❌ Policy Validation Failure: %v\n", err)
			return
		}

		fmt.Println("✅ Verification passed! Contract is secure and ready to deploy.")
	},
}

func init() {
	verifyCmd.Flags().StringVarP(&verifyFile, "file", "f", "", "Path to the intent contract file to check")
	rootCmd.AddCommand(verifyCmd)
}