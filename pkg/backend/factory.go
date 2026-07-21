package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ProjectConfig represents the structure of the company-wide nexus.yaml file
type ProjectConfig struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Backend    struct {
		Type   string `yaml:"type"`
		Bucket string `yaml:"bucket"`
		Region string `yaml:"region"`
	} `yaml:"backend"`
}

func NewBackend() (Backend, error) {
	// 1. Check if a company 'nexus.yaml' project file exists in the directory tree
	if cfg, err := discoverProjectConfig(); err == nil && cfg != nil {
		switch cfg.Backend.Type {
		case "s3":
			if cfg.Backend.Bucket == "" {
				return nil, fmt.Errorf("nexus.yaml specifies 's3' backend but missing 'bucket' field")
			}
			region := cfg.Backend.Region
			if region == "" {
				region = "us-east-1"
			}
			return NewS3Backend(cfg.Backend.Bucket, region)
		case "local":
			return NewLocalBackend()
		case "etcd":
			return NewEtcdBackend([]string{"127.0.0.1:2379"}, 5*time.Second)
		}
	}

	// 2. Fallback to environment variables if someone prefers them (or backwards compatibility)
	if backendType := os.Getenv("NEXUS_BACKEND"); backendType != "" {
		switch backendType {
		case "s3":
			bucket := os.Getenv("NEXUS_STATE_BUCKET")
			if bucket == "" {
				return nil, fmt.Errorf("NEXUS_STATE_BUCKET environment variable must be set")
			}
			region := os.Getenv("AWS_REGION")
			if region == "" {
				region = "us-east-1"
			}
			return NewS3Backend(bucket, region)
		case "etcd":
			return NewEtcdBackend([]string{"127.0.0.1:2379"}, 5*time.Second)
		case "local":
			return NewLocalBackend()
		}
	}

	// 3. Absolute default: Local File Backend (Just like Terraform!)
	return NewLocalBackend()
}

// discoverProjectConfig walks up directories looking for nexus.yaml
func discoverProjectConfig() (*ProjectConfig, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for {
		configPath := filepath.Join(dir, "nexus.yaml")
		if data, err := os.ReadFile(configPath); err == nil {
			var config ProjectConfig
			if err := yaml.Unmarshal(data, &config); err == nil && config.Kind == "ProjectConfig" {
				return &config, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root directory
		}
		dir = parent
	}

	return nil, fmt.Errorf("no nexus.yaml found")
}