package backend

import (
	"context"
)

type Backend interface {
	// State management
	FetchContracts(ctx context.Context, tenant string) (map[string][]byte, error)
	PutContract(ctx context.Context, tenant, name string, payload []byte) error
	
	// Distributed locking
	AcquireLock(ctx context.Context, tenant, environment, contractID, workerID string, ttlSeconds int) (string, bool, error)
	ReleaseLock(ctx context.Context, leaseID string) error
	ForceUnlock(ctx context.Context, tenant, environment, contractID string) error
	
	Close() error
}