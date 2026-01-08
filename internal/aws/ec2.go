package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"vaws/internal/model"
)

// ListEC2Instances lists EC2 instances, optionally filtered
func (c *Client) ListEC2Instances(ctx context.Context, filters ...types.Filter) ([]model.EC2Instance, error) {
	var instances []model.EC2Instance

	// Add running state filter by default
	filters = append(filters, types.Filter{
		Name:   aws.String("instance-state-name"),
		Values: []string{"running"},
	})

	paginator := ec2.NewDescribeInstancesPaginator(c.ec2, &ec2.DescribeInstancesInput{
		Filters: filters,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list EC2 instances: %w", err)
		}

		for _, reservation := range page.Reservations {
			for _, inst := range reservation.Instances {
				instances = append(instances, convertEC2Instance(inst))
			}
		}
	}

	return instances, nil
}

// GetEC2InstanceByName finds an EC2 instance by its Name tag
func (c *Client) GetEC2InstanceByName(ctx context.Context, name string) (*model.EC2Instance, error) {
	instances, err := c.ListEC2Instances(ctx, types.Filter{
		Name:   aws.String("tag:Name"),
		Values: []string{name},
	})
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no instance found with name: %s", name)
	}

	return &instances[0], nil
}

// GetEC2InstanceByID finds an EC2 instance by its instance ID
func (c *Client) GetEC2InstanceByID(ctx context.Context, instanceID string) (*model.EC2Instance, error) {
	out, err := c.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance %s: %w", instanceID, err)
	}

	for _, reservation := range out.Reservations {
		for _, inst := range reservation.Instances {
			instance := convertEC2Instance(inst)
			return &instance, nil
		}
	}

	return nil, fmt.Errorf("instance not found: %s", instanceID)
}

// FindEC2InstanceByTag finds an EC2 instance by a tag filter string (e.g., "Name=bastion")
func (c *Client) FindEC2InstanceByTag(ctx context.Context, tagFilter string) (*model.EC2Instance, error) {
	parts := strings.SplitN(tagFilter, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid tag filter format: %s (expected Key=Value)", tagFilter)
	}

	tagKey := parts[0]
	tagValue := parts[1]

	instances, err := c.ListEC2Instances(ctx, types.Filter{
		Name:   aws.String("tag:" + tagKey),
		Values: []string{tagValue},
	})
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no instance found with tag %s=%s", tagKey, tagValue)
	}

	return &instances[0], nil
}

// ListSSMManagedInstances lists EC2 instances that are managed by SSM
func (c *Client) ListSSMManagedInstances(ctx context.Context) ([]model.EC2Instance, error) {
	// First get all SSM managed instance IDs
	ssmInstanceIDs := make(map[string]bool)

	paginator := ssm.NewDescribeInstanceInformationPaginator(c.ssm, &ssm.DescribeInstanceInformationInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list SSM instances: %w", err)
		}

		for _, info := range page.InstanceInformationList {
			if info.InstanceId != nil && info.PingStatus == "Online" {
				ssmInstanceIDs[*info.InstanceId] = true
			}
		}
	}

	if len(ssmInstanceIDs) == 0 {
		return nil, nil
	}

	// Now get EC2 instance details for SSM-managed instances
	var instanceIDs []string
	for id := range ssmInstanceIDs {
		instanceIDs = append(instanceIDs, id)
	}

	// Query in batches of 100 (AWS limit)
	var instances []model.EC2Instance
	for i := 0; i < len(instanceIDs); i += 100 {
		end := i + 100
		if end > len(instanceIDs) {
			end = len(instanceIDs)
		}

		out, err := c.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: instanceIDs[i:end],
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe EC2 instances: %w", err)
		}

		for _, reservation := range out.Reservations {
			for _, inst := range reservation.Instances {
				instance := convertEC2Instance(inst)
				instance.SSMManaged = true
				instances = append(instances, instance)
			}
		}
	}

	return instances, nil
}

// FindJumpHost finds a suitable jump host for VPC access using configured settings.
// preferredVPCs is an optional list of VPC IDs to prefer (e.g., VPCs with execute-api endpoints).
func (c *Client) FindJumpHost(ctx context.Context, vpcID string, jumpHostConfig, jumpHostTagConfig string, defaultTags, defaultNames []string, preferredVPCs ...string) (*model.EC2Instance, error) {
	var triedMethods []string

	// Build a set of preferred VPCs for quick lookup
	preferredVPCSet := make(map[string]bool)
	for _, vpc := range preferredVPCs {
		preferredVPCSet[vpc] = true
	}

	// Priority 1: Configured jump host by name or ID
	if jumpHostConfig != "" {
		if strings.HasPrefix(jumpHostConfig, "i-") {
			inst, err := c.GetEC2InstanceByID(ctx, jumpHostConfig)
			if err == nil {
				return inst, nil
			}
			triedMethods = append(triedMethods, fmt.Sprintf("config instance ID '%s': %v", jumpHostConfig, err))
		} else {
			inst, err := c.GetEC2InstanceByName(ctx, jumpHostConfig)
			if err == nil {
				return inst, nil
			}
			triedMethods = append(triedMethods, fmt.Sprintf("config instance name '%s': %v", jumpHostConfig, err))
		}
	}

	// Priority 2: Configured jump host by tag
	if jumpHostTagConfig != "" {
		inst, err := c.FindEC2InstanceByTag(ctx, jumpHostTagConfig)
		if err == nil {
			return inst, nil
		}
		triedMethods = append(triedMethods, fmt.Sprintf("config tag '%s': %v", jumpHostTagConfig, err))
	}

	// Priority 3: Search by default tags in the VPC
	for _, tag := range defaultTags {
		inst, err := c.FindEC2InstanceByTag(ctx, tag)
		if err == nil {
			// If VPC is specified, verify the instance is in that VPC
			if vpcID != "" && inst.VpcID != vpcID {
				triedMethods = append(triedMethods, fmt.Sprintf("tag '%s': found but wrong VPC", tag))
				continue
			}
			return inst, nil
		}
		triedMethods = append(triedMethods, fmt.Sprintf("tag '%s': %v", tag, err))
	}

	// Priority 4: Search by default names
	for _, name := range defaultNames {
		inst, err := c.GetEC2InstanceByName(ctx, name)
		if err == nil {
			if vpcID != "" && inst.VpcID != vpcID {
				triedMethods = append(triedMethods, fmt.Sprintf("name '%s': found but wrong VPC", name))
				continue
			}
			return inst, nil
		}
		triedMethods = append(triedMethods, fmt.Sprintf("name '%s': %v", name, err))
	}

	// Priority 5: Get any SSM-managed instance
	// Prefer instances in VPCs with execute-api endpoints (preferredVPCs)
	ssmInstances, err := c.ListSSMManagedInstances(ctx)
	if err != nil {
		triedMethods = append(triedMethods, fmt.Sprintf("SSM instances: %v", err))
	} else if len(ssmInstances) == 0 {
		triedMethods = append(triedMethods, "SSM instances: none found online")
	} else {
		// First, try to find an instance in a preferred VPC
		if len(preferredVPCSet) > 0 {
			for _, inst := range ssmInstances {
				if preferredVPCSet[inst.VpcID] {
					return &inst, nil
				}
			}
			triedMethods = append(triedMethods, fmt.Sprintf("SSM instances: found %d but none in preferred VPCs", len(ssmInstances)))
		}

		// Then, try VPC filter if specified
		if vpcID != "" {
			for _, inst := range ssmInstances {
				if inst.VpcID == vpcID {
					return &inst, nil
				}
			}
			triedMethods = append(triedMethods, fmt.Sprintf("SSM instances: found %d but none in VPC %s", len(ssmInstances), vpcID))
		} else {
			// No VPC specified, return the first SSM-managed instance
			return &ssmInstances[0], nil
		}
	}

	return nil, fmt.Errorf("no suitable jump host found. Tried: %s", strings.Join(triedMethods, "; "))
}

// convertEC2Instance converts AWS EC2 instance to our model
func convertEC2Instance(inst types.Instance) model.EC2Instance {
	instance := model.EC2Instance{
		InstanceID:       aws.ToString(inst.InstanceId),
		InstanceType:     string(inst.InstanceType),
		State:            string(inst.State.Name),
		PrivateIPAddress: aws.ToString(inst.PrivateIpAddress),
		PublicIPAddress:  aws.ToString(inst.PublicIpAddress),
		VpcID:            aws.ToString(inst.VpcId),
		SubnetID:         aws.ToString(inst.SubnetId),
		LaunchTime:       aws.ToTime(inst.LaunchTime),
		Tags:             make(map[string]string),
	}

	// Extract tags
	for _, tag := range inst.Tags {
		key := aws.ToString(tag.Key)
		value := aws.ToString(tag.Value)
		instance.Tags[key] = value

		// Set name from Name tag
		if key == "Name" {
			instance.Name = value
		}
	}

	return instance
}
