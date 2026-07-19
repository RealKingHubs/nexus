package engine

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FormatContractFile reads a file, standardizes its indentation, and rewrites it in place
func FormatContractFile(filePath string) (bool, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("unable to read formatting target file: %w", err)
	}

	var rootNode yaml.Node
	if err := yaml.Unmarshal(fileBytes, &rootNode); err != nil {
		return false, fmt.Errorf("malformed syntax; cannot format a broken configuration file: %w", err)
	}

	var outBuffer bytes.Buffer
	encoder := yaml.NewEncoder(&outBuffer)
	encoder.SetIndent(2)

	if err := encoder.Encode(&rootNode); err != nil {
		return false, fmt.Errorf("failed to rewrite configuration node mappings: %w", err)
	}
	encoder.Close()

	formattedBytes := outBuffer.Bytes()
	if bytes.Equal(fileBytes, formattedBytes) {
		return false, nil
	}

	if err := os.WriteFile(filePath, formattedBytes, 0644); err != nil {
		return false, fmt.Errorf("failed writing formatted bytes to block storage: %w", err)
	}

	return true, nil
}

// CheckContractFileFormatting evaluates code structure without rewriting the file to disk.
// Returns true if the file needs formatting, false if it is already compliant.
func CheckContractFileFormatting(filePath string) (bool, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("unable to read target file for linting: %w", err)
	}

	var rootNode yaml.Node
	if err := yaml.Unmarshal(fileBytes, &rootNode); err != nil {
		return false, fmt.Errorf("malformed contract syntax: %w", err)
	}

	var outBuffer bytes.Buffer
	encoder := yaml.NewEncoder(&outBuffer)
	encoder.SetIndent(2) // Enforce canonical 2-space layout matrix
	_ = encoder.Encode(&rootNode)
	encoder.Close()

	// Return true if the memory arrays differ (indicating unformatted drift)
	return !bytes.Equal(fileBytes, outBuffer.Bytes()), nil
}