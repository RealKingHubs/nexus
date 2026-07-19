package registry

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdRegistry manages connections and distributed coordination states
type EtcdRegistry struct {
	client *clientv3.Client
}

// NewEtcdRegistry initializes a client connection pool to the live etcd cluster
func NewEtcdRegistry(endpoints []string, dialTimeout time.Duration) (*EtcdRegistry, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd nodes: %w", err)
	}

	return &EtcdRegistry{client: cli}, nil
}

// Close gracefully tears down backend network sockets
func (r *EtcdRegistry) Close() error {
	return r.client.Close()
}

// AcquireDistributedLock attempts to claim an exclusive concurrency lease fence for an environment
func (r *EtcdRegistry) AcquireDistributedLock(ctx context.Context, tenantID, envName, contract string, workerUUID string, ttlSeconds int64) (clientv3.LeaseID, bool, error) {
	// Build the simulated path key space string
	lockKey := fmt.Sprintf("_nexus/v1/tenants/%s/environments/%s/locks/%s", tenantID, envName, contract)

	// 1. Initialize a volatile Lease item with a dedicated survival lifetime boundary
	leaseResp, err := r.client.Grant(ctx, ttlSeconds)
	if err != nil {
		return 0, false, fmt.Errorf("failed to initialize infrastructure lease grant: %w", err)
	}
	leaseID := leaseResp.ID

	// 2. Build an atomic Compare-And-Swap (CAS) transaction block.
	// If the key version equals 0 (meaning the key path is completely vacant), commit the lock.
	txn := r.client.Txn(ctx).
		If(clientv3.Compare(clientv3.Version(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, workerUUID, clientv3.WithLease(leaseID))).
		Else(clientv3.OpGet(lockKey))

	txnResp, err := txn.Commit()
	if err != nil {
		_, _ = r.client.Revoke(ctx, leaseID) // Cleanup allocated lease space on failure
		return 0, false, err
	}

	// 3. Evaluate conditional verification status matrix returns
	if !txnResp.Succeeded {
		// Another processing node beat us to the transaction allocation race window
		_, _ = r.client.Revoke(ctx, leaseID)
		return 0, false, nil
	}

	// 4. Spawn a long-running background routine channel loop to extend the TTL while processing
	go func() {
		// Use a detached background context to ensure renewal continues even if the client request finishes early
		keepAliveChan, keepErr := r.client.KeepAlive(context.Background(), leaseID)
		if keepErr != nil {
			log.Printf("[Lock Engine Error] Failed to maintain heartbeat stream for lease %X: %v\n", leaseID, keepErr)
			return
		}

		// Drain the server tracking keep-alive stream responses to keep the pipeline open
		for range keepAliveChan {
			// Loop consumes packet headers continuously to maintain target lock integrity
		}
	}()

	return leaseID, true, nil
}

// ReleaseDistributedLock manually revokes a lease, instantly evicting the lock key space
func (r *EtcdRegistry) ReleaseDistributedLock(ctx context.Context, leaseID clientv3.LeaseID) error {
	_, err := r.client.Revoke(ctx, leaseID)
	if err != nil {
		return fmt.Errorf("failed to explicitly break infrastructure lease allocation block: %w", err)
	}
	return nil
}

// PutContract State stores the raw contract configuration payload inside etcd
func (r *EtcdRegistry) PutContract(ctx context.Context, tenantID, contractCode string, payload []byte) error {
	key := fmt.Sprintf("_nexus/v1/tenants/%s/contracts/%s", tenantID, contractCode)
	_, err := r.client.Put(ctx, key, string(payload))
	if err != nil {
		return fmt.Errorf("failed to commit contract state data to database: %w", err)
	}
	return nil
}

// FetchContractsByPrefix scans the etcd storage space for all records matching a prefix path
func (r *EtcdRegistry) FetchContractsByPrefix(ctx context.Context, tenantID string) (map[string][]byte, error) {
	prefix := fmt.Sprintf("_nexus/v1/tenants/%s/contracts/", tenantID)
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to scan prefix database nodes: %w", err)
	}

	results := make(map[string][]byte)
	for _, kv := range resp.Kvs {
		results[string(kv.Key)] = kv.Value
	}
	return results, nil
}

// GetContract retrieves a single contract's raw configuration bytes from etcd
func (r *EtcdRegistry) GetContract(ctx context.Context, tenantID, contractCode string) ([]byte, error) {
	key := fmt.Sprintf("_nexus/v1/tenants/%s/contracts/%s", tenantID, contractCode)
	
	resp, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contract from cluster database: %w", err)
	}

	// If the key doesn't exist, return nil without an error
	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	return resp.Kvs[0].Value, nil
}

// ForceUnlock removes an active lock key directly from etcd to clear a stalled pipeline
func (r *EtcdRegistry) ForceUnlock(ctx context.Context, tenantID, environment, contractCode string) error {
	// Reconstructing the specific key namespace used by the lock manager
	lockKey := fmt.Sprintf("_nexus/v1/locks/tenants/%s/environments/%s/contracts/%s", tenantID, environment, contractCode)
	
	_, err := r.client.Delete(ctx, lockKey)
	if err != nil {
		return fmt.Errorf("failed to forcefully delete environment lock node: %w", err)
	}
	return nil
}