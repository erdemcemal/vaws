package aws

import (
	"context"
	"fmt"
	"sort"
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
			service := convertService(svc)

			// Fetch container ports from task definition
			if svc.TaskDefinition != nil {
				containerPorts := c.getContainerPortsFromTaskDef(ctx, aws.ToString(svc.TaskDefinition))
				service.ContainerPorts = containerPorts
			}

			services = append(services, service)
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

// GetServicesForStack returns ECS services for clusters defined in a CloudFormation stack.
// It finds the cluster(s) from the stack and lists ALL services in those clusters.
func (c *Client) GetServicesForStack(ctx context.Context, stackName string) ([]model.Service, error) {
	log.Info("Getting ECS services for stack: %s", stackName)

	// Get ECS cluster(s) from stack resources
	clusterIDs, err := c.GetECSClustersFromStack(ctx, stackName)
	if err != nil {
		return nil, err
	}

	// If no clusters found in stack, try to infer from service ARNs
	if len(clusterIDs) == 0 {
		serviceIDs, err := c.GetECSServicesFromStack(ctx, stackName)
		if err != nil {
			return nil, err
		}
		// Extract cluster names from service ARNs
		clusterSet := make(map[string]bool)
		for _, svcID := range serviceIDs {
			if cluster := ExtractClusterFromServiceARN(svcID); cluster != "" {
				clusterSet[cluster] = true
			}
		}
		for cluster := range clusterSet {
			clusterIDs = append(clusterIDs, cluster)
		}
	}

	if len(clusterIDs) == 0 {
		log.Debug("No ECS clusters found in stack %s", stackName)
		return nil, nil
	}

	// List ALL services in each cluster (not just CF-defined ones)
	var allServices []model.Service
	for _, clusterID := range clusterIDs {
		services, err := c.ListServices(ctx, clusterID)
		if err != nil {
			log.Warn("Failed to list services in cluster %s: %v", clusterID, err)
			continue
		}
		allServices = append(allServices, services...)
	}

	// Sort services alphabetically by name (case-insensitive)
	sort.Slice(allServices, func(i, j int) bool {
		return strings.ToLower(allServices[i].Name) < strings.ToLower(allServices[j].Name)
	})

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

	// Cache for task definitions to avoid redundant API calls
	taskDefCache := make(map[string][]ecstypes.ContainerDefinition)

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

		// Get port mappings from task definition (for Fargate/awsvpc networking)
		taskDefARN := aws.ToString(t.TaskDefinitionArn)
		containerDefs, ok := taskDefCache[taskDefARN]
		if !ok && taskDefARN != "" {
			containerDefs = c.getContainerDefinitions(ctx, taskDefARN)
			taskDefCache[taskDefARN] = containerDefs
		}

		// Build a map of container name -> port mappings from task definition
		containerPortMap := make(map[string][]model.PortMapping)
		for _, cd := range containerDefs {
			name := aws.ToString(cd.Name)
			for _, pm := range cd.PortMappings {
				containerPortMap[name] = append(containerPortMap[name], model.PortMapping{
					ContainerPort: int(aws.ToInt32(pm.ContainerPort)),
					HostPort:      int(aws.ToInt32(pm.HostPort)),
					Protocol:      string(pm.Protocol),
					Name:          aws.ToString(pm.Name),
				})
			}
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

			// Add NetworkBindings (for EC2/bridge networking)
			for _, nb := range cont.NetworkBindings {
				container.NetworkBindings = append(container.NetworkBindings, model.NetworkBinding{
					ContainerPort: int(aws.ToInt32(nb.ContainerPort)),
					HostPort:      int(aws.ToInt32(nb.HostPort)),
					Protocol:      string(nb.Protocol),
				})
			}

			// Add PortMappings from task definition (for Fargate/awsvpc)
			if ports, ok := containerPortMap[container.Name]; ok {
				container.PortMappings = ports
			}

			task.Containers = append(task.Containers, container)
		}

		tasks = append(tasks, task)
	}

	log.Info("Found %d tasks for service %s", len(tasks), serviceName)
	return tasks, nil
}

// getContainerDefinitions fetches container definitions from a task definition.
func (c *Client) getContainerDefinitions(ctx context.Context, taskDefARN string) []ecstypes.ContainerDefinition {
	out, err := c.ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefARN),
	})
	if err != nil {
		log.Debug("Failed to describe task definition %s: %v", taskDefARN, err)
		return nil
	}
	if out.TaskDefinition == nil {
		return nil
	}
	return out.TaskDefinition.ContainerDefinitions
}

// getContainerPortsFromTaskDef fetches container names and their ports from a task definition.
func (c *Client) getContainerPortsFromTaskDef(ctx context.Context, taskDefARN string) []model.ContainerPort {
	containerDefs := c.getContainerDefinitions(ctx, taskDefARN)
	if containerDefs == nil {
		return nil
	}

	var containerPorts []model.ContainerPort
	for _, cd := range containerDefs {
		name := aws.ToString(cd.Name)
		var ports []int
		for _, pm := range cd.PortMappings {
			if port := int(aws.ToInt32(pm.ContainerPort)); port > 0 {
				ports = append(ports, port)
			}
		}
		if len(ports) > 0 {
			containerPorts = append(containerPorts, model.ContainerPort{
				ContainerName: name,
				Ports:         ports,
			})
		}
	}
	return containerPorts
}

// GetContainerLogConfigs extracts CloudWatch log configurations from a task definition.
// Returns log configs for containers using the awslogs driver.
func (c *Client) GetContainerLogConfigs(ctx context.Context, taskDefARN, taskID string) ([]model.ContainerLogConfig, error) {
	containerDefs := c.getContainerDefinitions(ctx, taskDefARN)
	if containerDefs == nil {
		return nil, fmt.Errorf("no container definitions found for task definition: %s", taskDefARN)
	}

	var configs []model.ContainerLogConfig

	for _, cd := range containerDefs {
		if cd.LogConfiguration == nil {
			continue
		}

		// Only support awslogs driver
		if cd.LogConfiguration.LogDriver != ecstypes.LogDriverAwslogs {
			continue
		}

		opts := cd.LogConfiguration.Options
		if opts == nil {
			continue
		}

		logGroup := opts["awslogs-group"]
		logStreamPrefix := opts["awslogs-stream-prefix"]
		logRegion := opts["awslogs-region"]

		if logGroup == "" || logStreamPrefix == "" {
			continue
		}

		containerName := aws.ToString(cd.Name)

		configs = append(configs, model.ContainerLogConfig{
			ContainerName:   containerName,
			LogGroup:        logGroup,
			LogStreamPrefix: logStreamPrefix,
			LogRegion:       logRegion,
			LogStreamName:   BuildLogStreamName(logStreamPrefix, containerName, taskID),
		})
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no containers with awslogs driver found")
	}

	return configs, nil
}

// GetSSMTarget returns the SSM target string for port forwarding to a Fargate task.
// Format: ecs:<cluster-name>_<task-id>_<runtime-id>
func GetSSMTarget(clusterName, taskID, runtimeID string) string {
	return fmt.Sprintf("ecs:%s_%s_%s", clusterName, taskID, runtimeID)
}
