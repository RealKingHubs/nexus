package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/nexus-io/nexus/pkg/engine"
)

type AWSProvider struct {
	client *ec2.Client
}

func NewAWSProvider(ctx context.Context, region string) (*AWSProvider, error) {
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS SDK configuration: %w", err)
	}

	client := ec2.NewFromConfig(cfg)
	return &AWSProvider{client: client}, nil
}

// 🔄 RECONCILE: Converges target EC2 instance to desired intent
func (a *AWSProvider) Reconcile(ctx context.Context, name string, spec engine.Spec) (engine.Status, error) {
	fmt.Printf("☁️ [AWS Provider] Checking EC2 instance with tag Name=%s...\n", name)

	// 1. Search for existing EC2 instances by Name tag
	describeInput := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{name},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"pending", "running", "stopping", "stopped"},
			},
		},
	}

	result, err := a.client.DescribeInstances(ctx, describeInput)
	if err != nil {
		return engine.Status{}, fmt.Errorf("failed to describe EC2 instances: %w", err)
	}

	var existingInstance *types.Instance
	for _, reservation := range result.Reservations {
		for _, inst := range reservation.Instances {
			instanceCopy := inst
			existingInstance = &instanceCopy
			break
		}
	}

	// 2. CREATE: Instance does not exist -> Provision new EC2 instance
	if existingInstance == nil {
		fmt.Printf("🚀 [AWS Provider] Creating new EC2 instance '%s'...\n", name)

		amiID := spec.Image
		if amiID == "" {
			amiID = "ami-0c7217cdde317cfec" // Default Amazon Linux 2023 in us-east-1
		}

		instanceType := types.InstanceTypeT2Micro
		if spec.InstanceType != "" {
			instanceType = types.InstanceType(spec.InstanceType)
		}

		runInput := &ec2.RunInstancesInput{
			ImageId:      aws.String(amiID),
			InstanceType: instanceType,
			MinCount:     aws.Int32(1),
			MaxCount:     aws.Int32(1),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeInstance,
					Tags: []types.Tag{
						{
							Key:   aws.String("Name"),
							Value: aws.String(name),
						},
						{
							Key:   aws.String("ManagedBy"),
							Value: aws.String("Nexus-Control-Plane"),
						},
					},
				},
			},
		}

		runResult, err := a.client.RunInstances(ctx, runInput)
		if err != nil {
			return engine.Status{}, fmt.Errorf("failed to launch EC2 instance: %w", err)
		}

		inst := runResult.Instances[0]
		return engine.Status{
			Phase: "Provisioning",
			Outputs: map[string]string{
				"instance_id":   aws.ToString(inst.InstanceId),
				"instance_type": string(inst.InstanceType),
				"public_ip":     aws.ToString(inst.PublicIpAddress),
				"provider":      "aws",
			},
		}, nil
	}

	instanceID := aws.ToString(existingInstance.InstanceId)
	stateName := string(existingInstance.State.Name)

	// 3. DRIFT HEALING: If instance is stopped, start it back up
	if stateName == "stopped" {
		fmt.Printf("🔄 [AWS Provider] Drift detected! Starting stopped EC2 instance %s...\n", instanceID)
		_, err := a.client.StartInstances(ctx, &ec2.StartInstancesInput{
			InstanceIds: []string{instanceID},
		})
		if err != nil {
			return engine.Status{}, fmt.Errorf("failed to restart stopped instance %s: %w", instanceID, err)
		}
		stateName = "starting"
	}

	fmt.Printf("🟢 [AWS Provider] EC2 Instance %s is active (State: %s)\n", instanceID, stateName)

	return engine.Status{
		Phase: "Running",
		Outputs: map[string]string{
			"instance_id":   instanceID,
			"instance_type": string(existingInstance.InstanceType),
			"public_ip":     aws.ToString(existingInstance.PublicIpAddress),
			"state":         stateName,
			"provider":      "aws",
		},
	}, nil
}

// 💥 DESTROY: Terminates matching EC2 instances
func (a *AWSProvider) Destroy(ctx context.Context, name string, spec engine.Spec) error {
	fmt.Printf("💥 [AWS Provider] Searching for EC2 instance '%s' to terminate...\n", name)

	describeInput := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{name},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"pending", "running", "stopped"},
			},
		},
	}

	result, err := a.client.DescribeInstances(ctx, describeInput)
	if err != nil {
		return fmt.Errorf("failed to describe instance for termination: %w", err)
	}

	var instanceIDs []string
	for _, reservation := range result.Reservations {
		for _, inst := range reservation.Instances {
			instanceIDs = append(instanceIDs, aws.ToString(inst.InstanceId))
		}
	}

	if len(instanceIDs) == 0 {
		fmt.Printf("ℹ️ [AWS Provider] No active EC2 instances found matching Name=%s.\n", name)
		return nil
	}

	fmt.Printf("🔥 [AWS Provider] Terminating EC2 instances: %v...\n", instanceIDs)
	_, err = a.client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: instanceIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to terminate instances %v: %w", instanceIDs, err)
	}

	fmt.Println("✨ [AWS Provider] EC2 termination signal dispatched successfully.")
	return nil
}