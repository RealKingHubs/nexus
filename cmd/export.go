package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nexus-io/nexus/pkg/backend"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var exportOutputFile string

var exportCmd = &cobra.Command{
	Use:     "export",
	GroupID: "core",
	Short:   "Export all managed intent contracts and cluster state for backup",
	Long:    `Queries the state backend registry for all active intent contracts and dumps them into a unified backup manifest file for disaster recovery and auditing.`,
	Example: `  nexus export -o cluster-backup.yaml
  nexus export`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔌 Initializing state backend interface to export cluster state...")

		bk, err := backend.NewBackend()
		if err != nil {
			fmt.Printf("❌ Backend Initialization Error: %v\n", err)
			return
		}
		defer bk.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		records, err := bk.FetchContracts(ctx, "default-tenant")
		if err != nil {
			fmt.Printf("❌ Failed to fetch contracts from state backend: %v\n", err)
			return
		}

		if len(records) == 0 {
			fmt.Println("🍁 No active intent contracts found in the state backend to export.")
			return
		}

		// Construct unified backup manifest payload structure
		backupData := map[string]interface{}{
			"apiVersion": "nexus.io/v1alpha1",
			"kind":       "ClusterBackup",
			"timestamp":  time.Now().Format(time.RFC3339),
			"tenant":     "default-tenant",
			"count":      len(records),
			"contracts":  map[string]interface{}{},
		}

		contractsMap := backupData["contracts"].(map[string]interface{})
		for name, payloadBytes := range records {
			var rawNode interface{}
			if err := yaml.Unmarshal(payloadBytes, &rawNode); err != nil {
				// Fallback to storing as raw string if unmarshaling fails
				contractsMap[name] = string(payloadBytes)
				continue
			}
			contractsMap[name] = rawNode
		}

		outBytes, err := yaml.Marshal(backupData)
		if err != nil {
			fmt.Printf("❌ Failed to encode backup manifest: %v\n", err)
			return
		}

		// Set default output filename with timestamp if not provided via flag
		if exportOutputFile == "" {
			timestampStr := time.Now().Format("20060102-150405")
			exportOutputFile = fmt.Sprintf("nexus-backup-%s.yaml", timestampStr)
		}

		if err := os.WriteFile(exportOutputFile, outBytes, 0644); err != nil {
			fmt.Printf("❌ Failed to write backup file: %v\n", err)
			return
		}

		fmt.Printf("✨ Cluster state successfully exported to: %s\n", exportOutputFile)
		fmt.Printf("📦 Total managed contracts backed up: %d\n", len(records))
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutputFile, "output", "o", "", "Destination file path for the backup manifest (defaults to nexus-backup-<timestamp>.yaml)")
	rootCmd.AddCommand(exportCmd)
}