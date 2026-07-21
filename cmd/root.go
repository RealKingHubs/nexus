package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nexus",
	Short: "Intent-driven, cloud-native infrastructure control plane",
	Long: `⚡ NEXUS CONTROL PLANE ENGINE
========================================================================
Nexus is an intent-driven control plane CLI that provisions, monitors, 
and continuously self-heals infrastructure workloads across Docker and 
cloud providers using etcd state backends.`,
	SilenceErrors: true, // Prevents duplicate error output since Execute() handles printing
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// 1. Organize commands into distinct visual groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "core",
		Title: "📦 CORE INFRASTRUCTURE OPERATIONS:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "daemon",
		Title: "🔄 AUTOMATION & SELF-HEALING DAEMON:",
	})
}