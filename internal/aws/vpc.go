package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"vaws/internal/model"
)

// ListVpcEndpoints lists VPC endpoints, optionally filtered by VPC ID
func (c *Client) ListVpcEndpoints(ctx context.Context, vpcID string) ([]model.VpcEndpoint, error) {
	var endpoints []model.VpcEndpoint

	input := &ec2.DescribeVpcEndpointsInput{}
	if vpcID != "" {
		input.Filters = []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		}
	}

	paginator := ec2.NewDescribeVpcEndpointsPaginator(c.ec2, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list VPC endpoints: %w", err)
		}

		for _, ep := range page.VpcEndpoints {
			endpoints = append(endpoints, convertVpcEndpoint(ep))
		}
	}

	return endpoints, nil
}

// FindAPIGatewayVpcEndpoint finds the VPC endpoint for API Gateway in the specified VPC
func (c *Client) FindAPIGatewayVpcEndpoint(ctx context.Context, vpcID string) (*model.VpcEndpoint, error) {
	endpoints, err := c.ListVpcEndpoints(ctx, vpcID)
	if err != nil {
		return nil, err
	}

	// Look for execute-api endpoint
	for _, ep := range endpoints {
		if strings.Contains(ep.ServiceName, "execute-api") && ep.State == "available" {
			return &ep, nil
		}
	}

	return nil, fmt.Errorf("no execute-api VPC endpoint found in VPC %s", vpcID)
}

// ListAPIGatewayVpcEndpoints lists all execute-api VPC endpoints in the account.
// Returns a map of VPC ID -> VPC endpoint for quick lookup.
func (c *Client) ListAPIGatewayVpcEndpoints(ctx context.Context) (map[string]*model.VpcEndpoint, error) {
	// List all VPC endpoints (no VPC filter)
	endpoints, err := c.ListVpcEndpoints(ctx, "")
	if err != nil {
		return nil, err
	}

	// Filter for execute-api endpoints and build map
	result := make(map[string]*model.VpcEndpoint)
	for _, ep := range endpoints {
		if strings.Contains(ep.ServiceName, "execute-api") && ep.State == "available" {
			epCopy := ep // Copy to avoid pointer issues
			result[ep.VpcID] = &epCopy
		}
	}

	return result, nil
}

// GetVpcEndpointByID gets a specific VPC endpoint by ID
func (c *Client) GetVpcEndpointByID(ctx context.Context, endpointID string) (*model.VpcEndpoint, error) {
	out, err := c.ec2.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: []string{endpointID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPC endpoint %s: %w", endpointID, err)
	}

	if len(out.VpcEndpoints) == 0 {
		return nil, fmt.Errorf("VPC endpoint not found: %s", endpointID)
	}

	ep := convertVpcEndpoint(out.VpcEndpoints[0])
	return &ep, nil
}

// GetVPCIDForInstance returns the VPC ID for an EC2 instance
func (c *Client) GetVPCIDForInstance(ctx context.Context, instanceID string) (string, error) {
	instance, err := c.GetEC2InstanceByID(ctx, instanceID)
	if err != nil {
		return "", err
	}
	return instance.VpcID, nil
}

// convertVpcEndpoint converts AWS VPC endpoint to our model
func convertVpcEndpoint(ep types.VpcEndpoint) model.VpcEndpoint {
	endpoint := model.VpcEndpoint{
		VpcEndpointID:   aws.ToString(ep.VpcEndpointId),
		VpcID:           aws.ToString(ep.VpcId),
		ServiceName:     aws.ToString(ep.ServiceName),
		State:           string(ep.State),
		VpcEndpointType: string(ep.VpcEndpointType),
		DNSEntries:      make([]string, 0),
		SubnetIDs:       make([]string, 0),
	}

	// Extract DNS entries
	for _, dns := range ep.DnsEntries {
		if dns.DnsName != nil {
			endpoint.DNSEntries = append(endpoint.DNSEntries, *dns.DnsName)
		}
	}

	// Extract subnet IDs
	endpoint.SubnetIDs = append(endpoint.SubnetIDs, ep.SubnetIds...)

	return endpoint
}
