package backend

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/nexus-io/nexus/pkg/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdBackend struct {
	reg *registry.EtcdRegistry
}

func NewEtcdBackend(endpoints []string, timeout time.Duration) (*EtcdBackend, error) {
	reg, err := registry.NewEtcdRegistry(endpoints, timeout)
	if err != nil {
		return nil, err
	}
	return &EtcdBackend{reg: reg}, nil
}

func (e *EtcdBackend) FetchContracts(ctx context.Context, tenant string) (map[string][]byte, error) {
	return e.reg.FetchContractsByPrefix(ctx, tenant)
}

func (e *EtcdBackend) PutContract(ctx context.Context, tenant, name string, payload []byte) error {
	return e.reg.PutContract(ctx, tenant, name, payload)
}

func (e *EtcdBackend) AcquireLock(ctx context.Context, tenant, environment, contractID, workerID string, ttlSeconds int) (string, bool, error) {
	leaseID, acquired, err := e.reg.AcquireDistributedLock(ctx, tenant, environment, contractID, workerID, int64(ttlSeconds))
	if err != nil {
		return "", false, err
	}
	if !acquired {
		return "", false, nil
	}
	// Convert etcd clientv3.LeaseID (int64) to a string token for the backend interface
	return strconv.FormatInt(int64(leaseID), 10), true, nil
}

func (e *EtcdBackend) ReleaseLock(ctx context.Context, leaseIDStr string) error {
	if leaseIDStr == "" {
		return nil
	}
	leaseInt, err := strconv.ParseInt(leaseIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid lease ID format: %w", err)
	}
	return e.reg.ReleaseDistributedLock(ctx, clientv3.LeaseID(leaseInt))
}

func (e *EtcdBackend) ForceUnlock(ctx context.Context, tenant, environment, contractID string) error {
	return e.reg.ForceUnlock(ctx, tenant, environment, contractID)
}

func (e *EtcdBackend) Close() error {
	return e.reg.Close()
}