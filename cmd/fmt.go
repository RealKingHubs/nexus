package cmd

import (
	"fmt"
	"os"

	"github.com/nexus-io/nexus/pkg/engine"
	"github.com/spf13/cobra"
)

var (
	fmtFile   string
	checkOnly bool
)

var fmtCmd = &cobra.Command{
	Use:   "fmt",
	Short: "Format Nexus contract files to standard style",
	Long:  `Automatically normalizes spacing, removes hidden tabs, and establishes a uniform 2-space layout hierarchy.`,
	Run: func(cmd *cobra.Command, args []string) {
		if fmtFile == "" {
			fmt.Println("❌ Error: Specify a target file path via -f or --file")
			os.Exit(1)
		}

		// Mode A: CI/CD Pipeline Linting Check
		if checkOnly {
			needsFormat, err := engine.CheckContractFileFormatting(fmtFile)
			if err != nil {
				fmt.Printf("❌ Linting Error: %v\n", err)
				os.Exit(1)
			}

			if needsFormat {
				fmt.Printf("❌ Code Quality Failure: file '%s' is poorly formatted!\n", fmtFile)
				fmt.Println("   -> Run 'nexus fmt -f <file>' locally to fix formatting issues before submitting a PR.")
				os.Exit(1) // 👈 Exit code 1 completely kills a GitHub Actions job line automatically
			}

			fmt.Printf("🟩 Code Style Verified: '%s' matches system specification layouts perfectly.\n", fmtFile)
			return
		}

		// Mode B: Standard In-Place Rewriting Tool Execution
		fmt.Printf("🧹 Formatting layout syntax for: %s...\n", fmtFile)
		mutated, err := engine.FormatContractFile(fmtFile)
		if err != nil {
			fmt.Printf("❌ Formatting Failure: %v\n", err)
			os.Exit(1)
		}

		if mutated {
			fmt.Println("✨ Successfully rewrote and standardized file structure configurations.")
		} else {
			fmt.Println("⭐ File was already perfectly formatted. No adjustments required.")
		}
	},
}

func init() {
	fmtCmd.Flags().StringVarP(&fmtFile, "file", "f", "", "Path to the intent contract file to format")
	fmtCmd.Flags().BoolVar(&checkOnly, "check", false, "Verify formatting compliance without rewriting data to disk")
	rootCmd.AddCommand(fmtCmd)
}