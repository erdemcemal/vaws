package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"vaws/internal/log"
	"vaws/internal/model"
)

// ListStacks returns all CloudFormation stacks (excluding deleted ones).
func (c *Client) ListStacks(ctx context.Context) ([]model.Stack, error) {
	log.Debug("Listing CloudFormation stacks...")

	var stacks []model.Stack
	paginator := cloudformation.NewListStacksPaginator(c.cfn, &cloudformation.ListStacksInput{
		StackStatusFilter: []cftypes.StackStatus{
			cftypes.StackStatusCreateComplete,
			cftypes.StackStatusCreateFailed,
			cftypes.StackStatusCreateInProgress,
			cftypes.StackStatusDeleteFailed,
			cftypes.StackStatusDeleteInProgress,
			cftypes.StackStatusRollbackComplete,
			cftypes.StackStatusRollbackFailed,
			cftypes.StackStatusRollbackInProgress,
			cftypes.StackStatusUpdateComplete,
			cftypes.StackStatusUpdateInProgress,
			cftypes.StackStatusUpdateRollbackComplete,
			cftypes.StackStatusUpdateRollbackFailed,
			cftypes.StackStatusUpdateRollbackInProgress,
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list stacks: %w", err)
		}

		for _, s := range page.StackSummaries {
			stacks = append(stacks, model.Stack{
				Name:         aws.ToString(s.StackName),
				ID:           aws.ToString(s.StackId),
				Status:       model.StackStatus(s.StackStatus),
				StatusReason: aws.ToString(s.StackStatusReason),
				CreatedAt:    aws.ToTime(s.CreationTime),
				UpdatedAt:    aws.ToTime(s.LastUpdatedTime),
			})
		}
	}

	// Sort stacks alphabetically by name (case-insensitive)
	sort.Slice(stacks, func(i, j int) bool {
		return strings.ToLower(stacks[i].Name) < strings.ToLower(stacks[j].Name)
	})

	log.Info("Found %d CloudFormation stacks", len(stacks))
	return stacks, nil
}

// DescribeStack returns detailed information about a specific stack.
func (c *Client) DescribeStack(ctx context.Context, stackName string) (*model.Stack, error) {
	log.Debug("Describing stack: %s", stackName)

	out, err := c.cfn.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe stack %s: %w", stackName, err)
	}

	if len(out.Stacks) == 0 {
		return nil, fmt.Errorf("stack %s not found", stackName)
	}

	s := out.Stacks[0]

	stack := &model.Stack{
		Name:         aws.ToString(s.StackName),
		ID:           aws.ToString(s.StackId),
		Status:       model.StackStatus(s.StackStatus),
		StatusReason: aws.ToString(s.StackStatusReason),
		CreatedAt:    aws.ToTime(s.CreationTime),
		UpdatedAt:    aws.ToTime(s.LastUpdatedTime),
		Description:  aws.ToString(s.Description),
		Tags:         make(map[string]string),
	}

	for _, tag := range s.Tags {
		stack.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	for _, out := range s.Outputs {
		stack.Outputs = append(stack.Outputs, model.StackOutput{
			Key:         aws.ToString(out.OutputKey),
			Value:       aws.ToString(out.OutputValue),
			Description: aws.ToString(out.Description),
			ExportName:  aws.ToString(out.ExportName),
		})
	}

	for _, p := range s.Parameters {
		stack.Parameters = append(stack.Parameters, model.StackParameter{
			Key:   aws.ToString(p.ParameterKey),
			Value: aws.ToString(p.ParameterValue),
		})
	}

	return stack, nil
}

// GetStackResources returns resources for a stack, optionally filtered by type.
func (c *Client) GetStackResources(ctx context.Context, stackName string, resourceType string) ([]cftypes.StackResourceSummary, error) {
	log.Debug("Getting resources for stack: %s (type filter: %s)", stackName, resourceType)

	var resources []cftypes.StackResourceSummary
	paginator := cloudformation.NewListStackResourcesPaginator(c.cfn, &cloudformation.ListStackResourcesInput{
		StackName: aws.String(stackName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list stack resources: %w", err)
		}

		for _, r := range page.StackResourceSummaries {
			if resourceType == "" || aws.ToString(r.ResourceType) == resourceType {
				resources = append(resources, r)
			}
		}
	}

	return resources, nil
}

// GetECSServicesFromStack returns ECS service ARNs/names defined in a CloudFormation stack.
func (c *Client) GetECSServicesFromStack(ctx context.Context, stackName string) ([]string, error) {
	resources, err := c.GetStackResources(ctx, stackName, "AWS::ECS::Service")
	if err != nil {
		return nil, err
	}

	var services []string
	for _, r := range resources {
		// PhysicalResourceId for ECS services is the service ARN or name
		if id := aws.ToString(r.PhysicalResourceId); id != "" {
			services = append(services, id)
		}
	}

	log.Debug("Found %d ECS services in stack %s", len(services), stackName)
	return services, nil
}

// GetECSClustersFromStack returns ECS cluster ARNs/names defined in a CloudFormation stack.
func (c *Client) GetECSClustersFromStack(ctx context.Context, stackName string) ([]string, error) {
	resources, err := c.GetStackResources(ctx, stackName, "AWS::ECS::Cluster")
	if err != nil {
		return nil, err
	}

	var clusters []string
	for _, r := range resources {
		if id := aws.ToString(r.PhysicalResourceId); id != "" {
			clusters = append(clusters, id)
		}
	}

	log.Debug("Found %d ECS clusters in stack %s", len(clusters), stackName)
	return clusters, nil
}

// ExtractClusterFromServiceARN extracts the cluster name from an ECS service ARN.
// ARN format: arn:aws:ecs:region:account:service/cluster-name/service-name
func ExtractClusterFromServiceARN(serviceARN string) string {
	if !strings.HasPrefix(serviceARN, "arn:aws:ecs:") {
		return ""
	}
	parts := strings.Split(serviceARN, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// GetLambdaFunctionsFromStack returns Lambda function names/ARNs defined in a CloudFormation stack.
func (c *Client) GetLambdaFunctionsFromStack(ctx context.Context, stackName string) ([]string, error) {
	resources, err := c.GetStackResources(ctx, stackName, "AWS::Lambda::Function")
	if err != nil {
		return nil, err
	}

	var functions []string
	for _, r := range resources {
		if id := aws.ToString(r.PhysicalResourceId); id != "" {
			functions = append(functions, id)
		}
	}

	log.Debug("Found %d Lambda functions in stack %s", len(functions), stackName)
	return functions, nil
}

// GetAPIGatewaysFromStack returns API Gateway IDs defined in a CloudFormation stack.
// Returns both REST APIs (v1) and HTTP APIs (v2).
func (c *Client) GetAPIGatewaysFromStack(ctx context.Context, stackName string) (restAPIIDs []string, httpAPIIDs []string, err error) {
	// Get REST APIs (API Gateway v1)
	restResources, err := c.GetStackResources(ctx, stackName, "AWS::ApiGateway::RestApi")
	if err != nil {
		return nil, nil, err
	}
	for _, r := range restResources {
		if id := aws.ToString(r.PhysicalResourceId); id != "" {
			restAPIIDs = append(restAPIIDs, id)
		}
	}

	// Get HTTP APIs (API Gateway v2)
	httpResources, err := c.GetStackResources(ctx, stackName, "AWS::ApiGatewayV2::Api")
	if err != nil {
		return nil, nil, err
	}
	for _, r := range httpResources {
		if id := aws.ToString(r.PhysicalResourceId); id != "" {
			httpAPIIDs = append(httpAPIIDs, id)
		}
	}

	log.Debug("Found %d REST APIs and %d HTTP APIs in stack %s", len(restAPIIDs), len(httpAPIIDs), stackName)
	return restAPIIDs, httpAPIIDs, nil
}
