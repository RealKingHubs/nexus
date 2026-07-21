package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type LocalBackend struct {
	stateDir string
}

type LocalStatePayload struct {
	Contracts map[string]string `json:"contracts"`
}

func NewLocalBackend() (*LocalBackend, error) {
	stateDir := ".nexus"
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create local state directory: %w", err)
	}
	return &LocalBackend{stateDir: stateDir}, nil
}

func (l *LocalBackend) getStateFilePath(tenant string) string {
	return filepath.Join(l.stateDir, fmt.Sprintf("state-%s.json", tenant))
}

func (l *LocalBackend) FetchContracts(ctx context.Context, tenant string) (map[string][]byte, error) {
	filePath := l.getStateFilePath(tenant)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string][]byte{}, nil
		}
		return nil, err
	}

	var state LocalStatePayload
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for name, yamlStr := range state.Contracts {
		result[name] = []byte(yamlStr)
	}
	return result, nil
}

func (l *LocalBackend) PutContract(ctx context.Context, tenant, name string, payload []byte) error {
	contracts, err := l.FetchContracts(ctx, tenant)
	if err != nil {
		contracts = make(map[string][]byte)
	}

	contracts[name] = payload

	stringContracts := make(map[string]string)
	for k, v := range contracts {
		stringContracts[k] = string(v)
	}

	state := LocalStatePayload{Contracts: stringContracts}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	filePath := l.getStateFilePath(tenant)
	return os.WriteFile(filePath, data, 0644)
}

func (l *LocalBackend) AcquireLock(ctx context.Context, tenant, environment, contractID, workerID string, ttlSeconds int) (string, bool, error) {
	lockDir := filepath.Join(l.stateDir, "locks", environment)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return "", false, err
	}

	lockFile := filepath.Join(lockDir, fmt.Sprintf("%s.lock", contractID))

	// Check if lock file already exists
	if _, err := os.Stat(lockFile); err == nil {
		return "", false, nil // Already locked
	}

	// Create lock file
	lockData := map[string]interface{}{
		"worker_id":  workerID,
		"expires_at": time.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}
	data, _ := json.Marshal(lockData)

	if err := os.WriteFile(lockFile, data, 0644); err != nil {
		return "", false, err
	}

	return lockFile, true, nil
}

func (l *LocalBackend) ReleaseLock(ctx context.Context, leaseID string) error {
	if leaseID == "" {
		return nil
	}
	// leaseID is the file path of the lock
	return os.Remove(leaseID)
}

func (l *LocalBackend) ForceUnlock(ctx context.Context, tenant, environment, contractID string) error {
	lockFile := filepath.Join(l.stateDir, "locks", environment, fmt.Sprintf("%s.lock", contractID))
	return os.Remove(lockFile)
}

func (l *LocalBackend) Close() error {
	return nil
}