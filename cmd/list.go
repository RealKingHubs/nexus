package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/nexus-io/nexus/pkg/registry"
	"gopkg.in/yaml.v3" // Imported to decode stored bytes dynamically
	"github.com/spf13/cobra"
)

// Re-using a minimal internal structure definition to map list data out cleanly
type ListSpec struct {
	Provider string `yaml:"provider"`
	Region   string `yaml:"region"`
}
type ListContractLayout struct {
	Metadata struct {
		Name        string `yaml:"name"`
		Environment string `yaml:"environment"`
	} `yaml:"metadata"`
	Spec ListSpec `yaml:"spec"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all managed intent environments",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔌 Querying active states from Nexus control plane registry...")

		// 1. Establish connection to etcd container sandbox
		reg, err := registry.NewEtcdRegistry([]string{"127.0.0.1:2379"}, 5*time.Second)
		if err != nil {
			fmt.Printf("❌ Registry Connection Error: %v\n", err)
			return
		}
		defer reg.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 2. Fetch all raw contract payloads stored under our tenant space
		rawRecords, err := reg.FetchContractsByPrefix(ctx, "default-tenant")
		if err != nil {
			fmt.Printf("❌ Read Operation Failure: %v\n", err)
			return
		}

		if len(rawRecords) == 0 {
			fmt.Println("\n🍁 No active intent contracts found. System space is completely empty.")
			return
		}

		// 3. Print out a beautifully clean structural text header table matrix
		fmt.Println("")
		fmt.Printf("%-28s %-12s %-15s %-12s\n", "CONTRACT NAME", "ENVIRONMENT", "CLOUD PROVIDER", "REGION")
		fmt.Println("----------------------------------------------------------------------")

		// 4. Loop through database entries, parse their content structures, and display them
		for _, payloadBytes := range rawRecords {
			var parsedContract ListContractLayout
			if err := yaml.Unmarshal(payloadBytes, &parsedContract); err != nil {
				// Skip single broken entry nodes if data corruption happened
				continue
			}

			fmt.Printf("%-28s %-12s %-15s %-12s\n", 
				parsedContract.Metadata.Name,
				parsedContract.Metadata.Environment,
				parsedContract.Spec.Provider,
				parsedContract.Spec.Region,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}