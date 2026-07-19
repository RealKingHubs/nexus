package engine

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// (Keep all your existing structures: Metadata, BusinessObjectives, etc. from our previous step)
type Metadata struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
}
type Network struct {
	Type            string `yaml:"type"`
	ExposedPublicly bool   `yaml:"exposedPublicly"`
}
type Database struct {
	Engine    string `yaml:"engine"`
	Version   string `yaml:"version"`
	StorageGB int    `yaml:"storageGB"`
}
type InfrastructureContext struct {
	Provider string   `yaml:"provider"`
	Region   string   `yaml:"region"`
	Network  Network  `yaml:"network"`
	Database Database `yaml:"database"`
}
type IntentContractSpec struct {
	APIVersion string                `yaml:"apiVersion"`
	Kind       string                `yaml:"kind"`
	Metadata   Metadata              `yaml:"metadata"`
	Spec       InfrastructureContext `yaml:"spec"`
}

// VerifyContractFile parses and validates file syntax
func VerifyContractFile(filePath string) (IntentContractSpec, error) {
	var contract IntentContractSpec
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return contract, fmt.Errorf("target file '%s' does not exist", filePath)
	}
	if fileInfo.IsDir() {
		return contract, fmt.Errorf("path '%s' is a directory", filePath)
	}

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return contract, fmt.Errorf("failed to read file: %w", err)
	}

	err = yaml.Unmarshal(fileBytes, &contract)
	if err != nil {
		return contract, fmt.Errorf("invalid YAML syntax:\n👉 %w", err)
	}

	if contract.APIVersion == "" || contract.Kind == "" {
		return contract, fmt.Errorf("missing mandatory fields 'apiVersion' or 'kind'")
	}
	return contract, nil
}

// PrintExecutionPlan displays exactly what will be built based on intent data
func PrintExecutionPlan(contract IntentContractSpec) {
	fmt.Println("\n📋 Nexus Speculative Execution Plan")
	fmt.Println("==========================================================")
	fmt.Printf("Engine will deploy intent matrix [%s] into environment [%s]:\n\n", 
		contract.Metadata.Name, contract.Metadata.Environment)

	fmt.Println("➕ [AWS Provider Core] will create:")
	fmt.Printf("   + VPC Network (Type: %s, Public Internet Access: %t)\n", 
		contract.Spec.Network.Type, contract.Spec.Network.ExposedPublicly)
	fmt.Printf("   + Relational Database Cluster (Engine: %s v%s, Allocation: %d GB Storage)\n", 
		contract.Spec.Database.Engine, contract.Spec.Database.Version, contract.Spec.Database.StorageGB)
	fmt.Printf("   + Orchestration Placement Region: %s\n", contract.Spec.Region)
	fmt.Println("==========================================================")
	fmt.Println("Plan: 3 resources to add, 0 to alter, 0 to destroy.")
}