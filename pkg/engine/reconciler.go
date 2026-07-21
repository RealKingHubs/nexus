package engine

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Metadata defines the target namespace mapping descriptors
type Metadata struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
}

// Spec holds the desired state configuration directives chosen by the engineer
type Spec struct {
	Provider string `yaml:"provider"`
	Region   string `yaml:"region"`
}

// Status holds the live runtime output parameters returned back from active cloud providers
type Status struct {
	Phase     string            `yaml:"phase"`
	UpdatedAt string            `yaml:"updatedAt"`
	Outputs   map[string]string `yaml:"outputs"`
}

// IntentContract represents the unified document structure layout for the engine
type IntentContract struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
	Status     Status   `yaml:"status"` // The new live infrastructure data layer
}

// VerifyContractFile parses and checks the structural schema validity of a target YAML file
func VerifyContractFile(filePath string) (*IntentContract, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read contract file: %w", err)
	}

	var contract IntentContract
	if err := yaml.Unmarshal(fileBytes, &contract); err != nil {
		return nil, fmt.Errorf("malformed YAML schema structure: %w", err)
	}

	// Structural boundary checks
	if contract.APIVersion != "nexus-io/v1alpha1" || contract.Kind != "IntentContract" {
		return nil, fmt.Errorf("unsupported API configuration group or resource kind")
	}
	if contract.Metadata.Name == "" {
		return nil, fmt.Errorf("metadata.name parameter cannot be empty")
	}

	return &contract, nil
}

// PrintExecutionPlan draws out a clean blueprint preview matrix before deployment execution
func PrintExecutionPlan(c *IntentContract) {
	fmt.Println("\n📋 Nexus Speculative Execution Plan")
	fmt.Println("==========================================================")
	fmt.Printf("🏢 Resource Target:  %s\n", c.Metadata.Name)
	fmt.Printf("🌐 Environment:      %s\n", c.Metadata.Environment)
	fmt.Printf("☁️  Cloud Provider:   %s (%s)\n", c.Spec.Provider, c.Spec.Region)
	fmt.Println("----------------------------------------------------------")
	fmt.Println("➕ [CREATE] Direct cloud assets matching core intent spec.")
	fmt.Println("==========================================================")
}

// GenerateRuntimeStatus simulates cloud provider API responses upon successful provisioning
func GenerateRuntimeStatus() Status {
	liveOutputs := make(map[string]string)
	liveOutputs["instance_id"] = "i-0bc78d129fa03eefb"
	liveOutputs["public_ip"]   = "54.210.43.87"
	liveOutputs["dns_url"]     = "http://payment-lb-184920.us-east-1.elb.amazonaws.com"

	return Status{
		Phase:     "Deployed",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Outputs:   liveOutputs,
	}
}