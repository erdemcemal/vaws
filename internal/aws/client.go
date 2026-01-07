// Package aws provides AWS SDK client management and operations.
package aws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Client wraps AWS service clients for a specific profile/region.
type Client struct {
	cfg     aws.Config
	profile string
	region  string
	cfn     *cloudformation.Client
	ecs     *ecs.Client
	lambda  *lambda.Client
	apigw   *apigateway.Client
	apigwv2 *apigatewayv2.Client
	ec2     *ec2.Client
	ssm     *ssm.Client
	cwlogs  *cloudwatchlogs.Client
}

// NewClient creates a new AWS client using the specified profile.
// If profile is empty, uses the default credential chain.
// If region is empty, uses the region from the profile or default.
func NewClient(ctx context.Context, profile, region string) (*Client, error) {
	opts := []func(*config.LoadOptions) error{}

	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	if region == "" {
		region = cfg.Region
	}

	return &Client{
		cfg:     cfg,
		profile: profile,
		region:  region,
		cfn:     cloudformation.NewFromConfig(cfg),
		ecs:     ecs.NewFromConfig(cfg),
		lambda:  lambda.NewFromConfig(cfg),
		apigw:   apigateway.NewFromConfig(cfg),
		apigwv2: apigatewayv2.NewFromConfig(cfg),
		ec2:     ec2.NewFromConfig(cfg),
		ssm:     ssm.NewFromConfig(cfg),
		cwlogs:  cloudwatchlogs.NewFromConfig(cfg),
	}, nil
}

// Profile returns the configured profile name.
func (c *Client) Profile() string {
	return c.profile
}

// Region returns the configured region.
func (c *Client) Region() string {
	return c.region
}

// CloudFormation returns the CloudFormation client.
func (c *Client) CloudFormation() *cloudformation.Client {
	return c.cfn
}

// ECS returns the ECS client.
func (c *Client) ECS() *ecs.Client {
	return c.ecs
}

// Lambda returns the Lambda client.
func (c *Client) Lambda() *lambda.Client {
	return c.lambda
}

// APIGateway returns the API Gateway v1 (REST API) client.
func (c *Client) APIGateway() *apigateway.Client {
	return c.apigw
}

// APIGatewayV2 returns the API Gateway v2 (HTTP API) client.
func (c *Client) APIGatewayV2() *apigatewayv2.Client {
	return c.apigwv2
}

// EC2 returns the EC2 client.
func (c *Client) EC2() *ec2.Client {
	return c.ec2
}

// SSM returns the SSM client.
func (c *Client) SSM() *ssm.Client {
	return c.ssm
}

// CloudWatchLogs returns the CloudWatch Logs client.
func (c *Client) CloudWatchLogs() *cloudwatchlogs.Client {
	return c.cwlogs
}

// Config returns the underlying AWS config.
func (c *Client) Config() aws.Config {
	return c.cfg
}

// ListProfiles returns all available AWS profiles from the config file.
func ListProfiles() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".aws", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{"default"}, nil
		}
		return nil, fmt.Errorf("failed to read AWS config: %w", err)
	}

	var profiles []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
			name := strings.TrimPrefix(line, "[profile ")
			name = strings.TrimSuffix(name, "]")
			profiles = append(profiles, name)
		} else if line == "[default]" {
			profiles = append(profiles, "default")
		}
	}

	if len(profiles) == 0 {
		profiles = append(profiles, "default")
	}

	return profiles, nil
}
