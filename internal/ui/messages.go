package ui

import (
	"vaws/internal/aws"
	"vaws/internal/model"
)

// Messages for bubbletea.
type (
	// stacksLoadedMsg is sent when stacks are loaded.
	stacksLoadedMsg struct {
		stacks []model.Stack
		err    error
	}

	// servicesLoadedMsg is sent when services are loaded.
	servicesLoadedMsg struct {
		services []model.Service
		err      error
	}

	// functionsLoadedMsg is sent when Lambda functions are loaded.
	functionsLoadedMsg struct {
		functions []model.Function
		err       error
	}

	// restAPIsLoadedMsg is sent when REST APIs are loaded.
	restAPIsLoadedMsg struct {
		apis []model.RestAPI
		err  error
	}

	// httpAPIsLoadedMsg is sent when HTTP APIs are loaded.
	httpAPIsLoadedMsg struct {
		apis []model.HttpAPI
		err  error
	}

	// apiStagesLoadedMsg is sent when API stages are loaded.
	apiStagesLoadedMsg struct {
		stages []model.APIStage
		err    error
	}

	// tasksLoadedMsg is sent when tasks are loaded for a service.
	tasksLoadedMsg struct {
		service model.Service
		tasks   []model.Task
		err     error
	}

	// tasksLoadedMsgWithPort is sent when tasks are loaded with a custom port.
	tasksLoadedMsgWithPort struct {
		service   model.Service
		tasks     []model.Task
		err       error
		localPort int
	}

	// tasksLoadedMsgForRestart is sent when tasks are loaded for tunnel restart.
	tasksLoadedMsgForRestart struct {
		tunnelInfo model.Tunnel
		tasks      []model.Task
		err        error
	}

	// tunnelStartedMsg is sent when a tunnel is started.
	tunnelStartedMsg struct {
		tunnel *model.Tunnel
		err    error
	}

	// apiGWTunnelStartedMsg is sent when an API Gateway tunnel is started.
	apiGWTunnelStartedMsg struct {
		tunnel *model.APIGatewayTunnel
		err    error
	}

	// jumpHostFoundMsg is sent when a jump host is found for private API Gateway.
	jumpHostFoundMsg struct {
		jumpHost          *model.EC2Instance
		vpcEndpoint       *model.VpcEndpoint
		vpcEndpointErr    error
		stage             model.APIStage
		api               interface{}
		localPort         int
		err               error
		vpcsWithEndpoints []string // VPCs that have execute-api endpoints
	}

	// ec2InstancesLoadedMsg is sent when EC2 instances are loaded for jump host selection.
	ec2InstancesLoadedMsg struct {
		instances []model.EC2Instance
		err       error
	}

	// tunnelRefreshMsg triggers a refresh of the tunnel list.
	tunnelRefreshMsg struct{}

	// errMsg is sent when an error occurs.
	errMsg struct {
		err error
	}

	// clientCreatedMsg is sent when AWS client is created after profile selection.
	clientCreatedMsg struct {
		client *aws.Client
		err    error
	}

	// cloudWatchLogConfigsLoadedMsg is sent when log configs are loaded.
	cloudWatchLogConfigsLoadedMsg struct {
		configs []model.ContainerLogConfig
		service model.Service
		task    model.Task
		err     error
	}

	// cloudWatchLogsLoadedMsg is sent when CloudWatch logs are loaded.
	cloudWatchLogsLoadedMsg struct {
		entries       []model.CloudWatchLogEntry
		lastTimestamp int64
		err           error
	}

	// queuesLoadedMsg is sent when SQS queues are loaded.
	queuesLoadedMsg struct {
		queues []model.Queue
		err    error
	}

	// clustersLoadedMsg is sent when ECS clusters are loaded.
	clustersLoadedMsg struct {
		clusters []model.Cluster
		err      error
	}

	// lambdaInvocationResultMsg is sent when Lambda invocation completes.
	lambdaInvocationResultMsg struct {
		result *model.InvocationResult
		err    error
	}

	// regionChangedMsg is sent when AWS region is changed.
	regionChangedMsg struct {
		client *aws.Client
		region string
		err    error
	}
)
