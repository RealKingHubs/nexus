package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Backend struct {
	client     *s3.Client
	bucketName string
	region     string
}

type S3StatePayload struct {
	Contracts map[string]string `json:"contracts"` // name -> YAML string
}

type S3LockPayload struct {
	WorkerID  string    `json:"worker_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewS3Backend(bucketName, region string) (*S3Backend, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for S3 backend: %w", err)
	}

	return &S3Backend{
		client:     s3.NewFromConfig(cfg),
		bucketName: bucketName,
		region:     region,
	}, nil
}

func (s *S3Backend) FetchContracts(ctx context.Context, tenant string) (map[string][]byte, error) {
	key := fmt.Sprintf("tenants/%s/state.json", tenant)
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		// If state file doesn't exist yet, return empty map
		return map[string][]byte{}, nil
	}
	defer output.Body.Close()

	var state S3StatePayload
	if err := json.NewDecoder(output.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode S3 state JSON: %w", err)
	}

	result := make(map[string][]byte)
	for name, yamlStr := range state.Contracts {
		result[name] = []byte(yamlStr)
	}
	return result, nil
}

func (s *S3Backend) PutContract(ctx context.Context, tenant, name string, payload []byte) error {
	// Fetch existing state
	contracts, err := s.FetchContracts(ctx, tenant)
	if err != nil {
		contracts = make(map[string][]byte)
	}

	contracts[name] = payload

	stringContracts := make(map[string]string)
	for k, v := range contracts {
		stringContracts[k] = string(v)
	}

	state := S3StatePayload{Contracts: stringContracts}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state JSON: %w", err)
	}

	key := fmt.Sprintf("tenants/%s/state.json", tenant)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	return err
}

func (s *S3Backend) AcquireLock(ctx context.Context, tenant, environment, contractID, workerID string, ttlSeconds int) (string, bool, error) {
	lockKey := fmt.Sprintf("tenants/%s/locks/%s/%s.lock", tenant, environment, contractID)

	// Check if active lock exists
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(lockKey),
	})
	if err == nil {
		defer output.Body.Close()
		var lock S3LockPayload
		if json.NewDecoder(output.Body).Decode(&lock) == nil {
			if time.Now().Before(lock.ExpiresAt) {
				return "", false, nil // Already locked by another worker
			}
		}
	}

	// Create new lock
	lockData := S3LockPayload{
		WorkerID:  workerID,
		ExpiresAt: time.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}
	data, _ := json.Marshal(lockData)

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(lockKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return "", false, err
	}

	return lockKey, true, nil
}

func (s *S3Backend) ReleaseLock(ctx context.Context, leaseID string) error {
	if leaseID == "" {
		return nil
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(leaseID),
	})
	return err
}

func (s *S3Backend) ForceUnlock(ctx context.Context, tenant, environment, contractID string) error {
	lockKey := fmt.Sprintf("tenants/%s/locks/%s/%s.lock", tenant, environment, contractID)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(lockKey),
	})
	return err
}

func (s *S3Backend) Close() error {
	return nil
}