package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"vaws/internal/log"
	"vaws/internal/model"
)

// ListClusters returns all ECS clusters.
func (c *Client) ListClusters(ctx context.Context) ([]model.Cluster, error) {
	log.Debug("Listing ECS clusters...")

	var clusterARNs []string
	paginator := ecs.NewListClustersPaginator(c.ecs, &ecs.ListClustersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}
		clusterARNs = append(clusterARNs, page.ClusterArns...)
	}

	if len(clusterARNs) == 0 {
		return nil, nil
	}

	// Describe clusters to get details
	out, err := c.ecs.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: clusterARNs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe clusters: %w", err)
	}

	var clusters []model.Cluster
	for _, cl := range out.Clusters {
		clusters = append(clusters, model.Cluster{
			Name:                              aws.ToString(cl.ClusterName),
			ARN:                               aws.ToString(cl.ClusterArn),
			Status:                            aws.ToString(cl.Status),
			ActiveServicesCount:               int(cl.ActiveServicesCount),
			RunningTasksCount:                 int(cl.RunningTasksCount),
			PendingTasksCount:                 int(cl.PendingTasksCount),
			RegisteredContainerInstancesCount: int(cl.RegisteredContainerInstancesCount),
		})
	}

	log.Info("Found %d ECS clusters", len(clusters))
	return clusters, nil
}

// ListServices returns all ECS services in a cluster.
func (c *Client) ListServices(ctx context.Context, clusterARN string) ([]model.Service, error) {
	log.Debug("Listing ECS services in cluster: %s", clusterARN)

	var serviceARNs []string
	paginator := ecs.NewListServicesPaginator(c.ecs, &ecs.ListServicesInput{
		Cluster: aws.String(clusterARN),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list services: %w", err)
		}
		serviceARNs = append(serviceARNs, page.ServiceArns...)
	}

	if len(serviceARNs) == 0 {
		return nil, nil
	}

	return c.DescribeServices(ctx, clusterARN, serviceARNs)
}

// DescribeServices returns detailed information about specific services.
func (c *Client) DescribeServices(ctx context.Context, clusterARN string, serviceARNs []string) ([]model.Service, error) {
	if len(serviceARNs) == 0 {
		return nil, nil
	}

	// DescribeServices has a limit of 10 services per call
	var services []model.Service
	for i := 0; i < len(serviceARNs); i += 10 {
		end := i + 10
		if end > len(serviceARNs) {
			end = len(serviceARNs)
		}

		batch := serviceARNs[i:end]
		out, err := c.ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  aws.String(clusterARN),
			Services: batch,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe services: %w", err)
		}

		for _, svc := range out.Services {
			services = append(services, convertService(svc))
		}
	}

	log.Debug("Described %d ECS services", len(services))
	return services, nil
}

// DescribeService returns detailed information about a specific service.
func (c *Client) DescribeService(ctx context.Context, clusterARN, serviceName string) (*model.Service, error) {
	log.Debug("Describing ECS service: %s in cluster %s", serviceName, clusterARN)

	out, err := c.ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterARN),
		Services: []string{serviceName},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe service %s: %w", serviceName, err)
	}

	if len(out.Services) == 0 {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	svc := convertService(out.Services[0])
	return &svc, nil
}

// GetServicesForStack returns ECS services that belong to a CloudFormation stack.
// It finds services by looking at stack resources.
func (c *Client) GetServicesForStack(ctx context.Context, stackName string) ([]model.Service, error) {
	log.Info("Getting ECS services for stack: %s", stackName)

	// Get ECS service ARNs from stack resources
	serviceIDs, err := c.GetECSServicesFromStack(ctx, stackName)
	if err != nil {
		return nil, err
	}

	if len(serviceIDs) == 0 {
		log.Debug("No ECS services found in stack %s", stackName)
		return nil, nil
	}

	// Group services by cluster
	servicesByCluster := make(map[string][]string)
	for _, svcID := range serviceIDs {
		var cluster string
		if strings.Contains(svcID, "/") {
			// ARN format: arn:aws:ecs:region:account:service/cluster-name/service-name
			cluster = ExtractClusterFromServiceARN(svcID)
		}
		if cluster == "" {
			// Try to infer cluster from stack's ECS cluster resources
			clusters, _ := c.GetECSClustersFromStack(ctx, stackName)
			if len(clusters) > 0 {
				cluster = clusters[0]
			}
		}
		if cluster != "" {
			servicesByCluster[cluster] = append(servicesByCluster[cluster], svcID)
		}
	}

	// Describe all services
	var allServices []model.Service
	for cluster, svcARNs := range servicesByCluster {
		services, err := c.DescribeServices(ctx, cluster, svcARNs)
		if err != nil {
			log.Warn("Failed to describe services in cluster %s: %v", cluster, err)
			continue
		}
		allServices = append(allServices, services...)
	}

	log.Info("Found %d ECS services for stack %s", len(allServices), stackName)
	return allServices, nil
}

func convertService(svc ecstypes.Service) model.Service {
	service := model.Service{
		Name:                 aws.ToString(svc.ServiceName),
		ARN:                  aws.ToString(svc.ServiceArn),
		ClusterARN:           aws.ToString(svc.ClusterArn),
		Status:               model.ServiceStatus(aws.ToString(svc.Status)),
		DesiredCount:         int(svc.DesiredCount),
		RunningCount:         int(svc.RunningCount),
		PendingCount:         int(svc.PendingCount),
		TaskDefinition:       aws.ToString(svc.TaskDefinition),
		LaunchType:           string(svc.LaunchType),
		CreatedAt:            aws.ToTime(svc.CreatedAt),
		EnableExecuteCommand: svc.EnableExecuteCommand,
	}

	// Extract cluster name from ARN
	if service.ClusterARN != "" {
		parts := strings.Split(service.ClusterARN, "/")
		if len(parts) > 0 {
			service.ClusterName = parts[len(parts)-1]
		}
	}

	for _, d := range svc.Deployments {
		service.Deployments = append(service.Deployments, model.Deployment{
			ID:             aws.ToString(d.Id),
			Status:         aws.ToString(d.Status),
			DesiredCount:   int(d.DesiredCount),
			RunningCount:   int(d.RunningCount),
			PendingCount:   int(d.PendingCount),
			TaskDefinition: aws.ToString(d.TaskDefinition),
			CreatedAt:      aws.ToTime(d.CreatedAt),
			UpdatedAt:      aws.ToTime(d.UpdatedAt),
		})
	}

	return service
}

// ListTasksForService returns running tasks for a service.
func (c *Client) ListTasksForService(ctx context.Context, clusterARN, serviceName string) ([]model.Task, error) {
	log.Debug("Listing tasks for service: %s in cluster %s", serviceName, clusterARN)

	// List task ARNs
	listOut, err := c.ecs.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:     aws.String(clusterARN),
		ServiceName: aws.String(serviceName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(listOut.TaskArns) == 0 {
		return nil, nil
	}

	// Describe tasks to get details
	descOut, err := c.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(clusterARN),
		Tasks:   listOut.TaskArns,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe tasks: %w", err)
	}

	var tasks []model.Task
	for _, t := range descOut.Tasks {
		task := model.Task{
			TaskARN:           aws.ToString(t.TaskArn),
			ClusterARN:        aws.ToString(t.ClusterArn),
			TaskDefinitionARN: aws.ToString(t.TaskDefinitionArn),
			LastStatus:        aws.ToString(t.LastStatus),
			DesiredStatus:     aws.ToString(t.DesiredStatus),
			LaunchType:        string(t.LaunchType),
			StartedAt:         aws.ToTime(t.StartedAt),
		}

		// Extract task ID from ARN
		// ARN format: arn:aws:ecs:region:account:task/cluster-name/task-id
		if parts := strings.Split(task.TaskARN, "/"); len(parts) > 0 {
			task.TaskID = parts[len(parts)-1]
		}

		// Get container details
		for _, cont := range t.Containers {
			container := model.Container{
				Name:         aws.ToString(cont.Name),
				ContainerARN: aws.ToString(cont.ContainerArn),
				RuntimeID:    aws.ToString(cont.RuntimeId),
				LastStatus:   aws.ToString(cont.LastStatus),
				Image:        aws.ToString(cont.Image),
			}

			for _, nb := range cont.NetworkBindings {
				container.NetworkBindings = append(container.NetworkBindings, model.NetworkBinding{
					ContainerPort: int(aws.ToInt32(nb.ContainerPort)),
					HostPort:      int(aws.ToInt32(nb.HostPort)),
					Protocol:      string(nb.Protocol),
				})
			}

			task.Containers = append(task.Containers, container)
		}

		tasks = append(tasks, task)
	}

	log.Info("Found %d tasks for service %s", len(tasks), serviceName)
	return tasks, nil
}

// GetSSMTarget returns the SSM target string for port forwarding to a Fargate task.
// Format: ecs:<cluster-name>_<task-id>_<runtime-id>
func GetSSMTarget(clusterName, taskID, runtimeID string) string {
	return fmt.Sprintf("ecs:%s_%s_%s", clusterName, taskID, runtimeID)
}
