package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	debug   bool // Changed from verbose to debug
)

var rootCmd = &cobra.Command{
	Use:     "nexus",
	Short:   "Nexus is an intent-driven cloud-native control plane engine",
	Version: "v0.1.0-alpha", // 👈 Adding this enables --version and -v automatically!
	Long: `A high-performance infrastructure orchestration plane written in Go 
that continuously observes, analyzes, and self-heals cloud states based on 
declarative intent contracts.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Execution failure: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.nexus.yaml)")
	// 👈 Changed shorthand to "-d" so "-v" belongs to version!
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable fine-grained debug telemetry logging")
}