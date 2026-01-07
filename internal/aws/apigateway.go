package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"

	"vaws/internal/model"
)

// ListRestAPIs lists all REST APIs (API Gateway v1).
func (c *Client) ListRestAPIs(ctx context.Context) ([]model.RestAPI, error) {
	var apis []model.RestAPI

	paginator := apigateway.NewGetRestApisPaginator(c.apigw, &apigateway.GetRestApisInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list REST APIs: %w", err)
		}

		for _, api := range page.Items {
			// Get endpoint type and VPC endpoint IDs
			endpointType := "REGIONAL"
			var vpcEndpointIds []string
			if api.EndpointConfiguration != nil {
				if len(api.EndpointConfiguration.Types) > 0 {
					endpointType = string(api.EndpointConfiguration.Types[0])
				}
				vpcEndpointIds = api.EndpointConfiguration.VpcEndpointIds
			}

			apis = append(apis, model.RestAPI{
				ID:             aws.ToString(api.Id),
				Name:           aws.ToString(api.Name),
				Description:    aws.ToString(api.Description),
				CreatedDate:    aws.ToTime(api.CreatedDate),
				EndpointType:   endpointType,
				Version:        aws.ToString(api.Version),
				VpcEndpointIds: vpcEndpointIds,
			})
		}
	}

	return apis, nil
}

// GetRestAPI returns a single REST API by ID.
func (c *Client) GetRestAPI(ctx context.Context, apiID string) (*model.RestAPI, error) {
	out, err := c.apigw.GetRestApi(ctx, &apigateway.GetRestApiInput{
		RestApiId: aws.String(apiID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get REST API %s: %w", apiID, err)
	}

	endpointType := "REGIONAL"
	var vpcEndpointIds []string
	if out.EndpointConfiguration != nil {
		if len(out.EndpointConfiguration.Types) > 0 {
			endpointType = string(out.EndpointConfiguration.Types[0])
		}
		vpcEndpointIds = out.EndpointConfiguration.VpcEndpointIds
	}

	return &model.RestAPI{
		ID:             aws.ToString(out.Id),
		Name:           aws.ToString(out.Name),
		Description:    aws.ToString(out.Description),
		CreatedDate:    aws.ToTime(out.CreatedDate),
		EndpointType:   endpointType,
		Version:        aws.ToString(out.Version),
		VpcEndpointIds: vpcEndpointIds,
	}, nil
}

// GetRestAPIStages returns the stages for a REST API.
func (c *Client) GetRestAPIStages(ctx context.Context, apiID string) ([]model.APIStage, error) {
	out, err := c.apigw.GetStages(ctx, &apigateway.GetStagesInput{
		RestApiId: aws.String(apiID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get stages for REST API %s: %w", apiID, err)
	}

	var stages []model.APIStage
	for _, s := range out.Item {
		invokeURL := fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/%s",
			apiID, c.region, aws.ToString(s.StageName))

		stages = append(stages, model.APIStage{
			Name:         aws.ToString(s.StageName),
			Description:  aws.ToString(s.Description),
			DeploymentID: aws.ToString(s.DeploymentId),
			CreatedDate:  aws.ToTime(s.CreatedDate),
			LastUpdated:  aws.ToTime(s.LastUpdatedDate),
			InvokeURL:    invokeURL,
		})
	}

	return stages, nil
}

// ListHttpAPIs lists all HTTP APIs (API Gateway v2).
func (c *Client) ListHttpAPIs(ctx context.Context) ([]model.HttpAPI, error) {
	var apis []model.HttpAPI

	var nextToken *string
	for {
		out, err := c.apigwv2.GetApis(ctx, &apigatewayv2.GetApisInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list HTTP APIs: %w", err)
		}

		for _, api := range out.Items {
			apis = append(apis, model.HttpAPI{
				ID:           aws.ToString(api.ApiId),
				Name:         aws.ToString(api.Name),
				Description:  aws.ToString(api.Description),
				ProtocolType: string(api.ProtocolType),
				CreatedDate:  aws.ToTime(api.CreatedDate),
				ApiEndpoint:  aws.ToString(api.ApiEndpoint),
				Version:      aws.ToString(api.Version),
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	return apis, nil
}

// GetHttpAPI returns a single HTTP API by ID.
func (c *Client) GetHttpAPI(ctx context.Context, apiID string) (*model.HttpAPI, error) {
	out, err := c.apigwv2.GetApi(ctx, &apigatewayv2.GetApiInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP API %s: %w", apiID, err)
	}

	return &model.HttpAPI{
		ID:           aws.ToString(out.ApiId),
		Name:         aws.ToString(out.Name),
		Description:  aws.ToString(out.Description),
		ProtocolType: string(out.ProtocolType),
		CreatedDate:  aws.ToTime(out.CreatedDate),
		ApiEndpoint:  aws.ToString(out.ApiEndpoint),
		Version:      aws.ToString(out.Version),
	}, nil
}

// GetHttpAPIStages returns the stages for an HTTP API.
func (c *Client) GetHttpAPIStages(ctx context.Context, apiID string) ([]model.APIStage, error) {
	out, err := c.apigwv2.GetStages(ctx, &apigatewayv2.GetStagesInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get stages for HTTP API %s: %w", apiID, err)
	}

	var stages []model.APIStage
	for _, s := range out.Items {
		stages = append(stages, model.APIStage{
			Name:         aws.ToString(s.StageName),
			Description:  aws.ToString(s.Description),
			DeploymentID: aws.ToString(s.DeploymentId),
			CreatedDate:  aws.ToTime(s.CreatedDate),
			LastUpdated:  aws.ToTime(s.LastUpdatedDate),
		})
	}

	return stages, nil
}

// GetHttpAPIRoutes returns the routes for an HTTP API.
func (c *Client) GetHttpAPIRoutes(ctx context.Context, apiID string) ([]model.APIRoute, error) {
	out, err := c.apigwv2.GetRoutes(ctx, &apigatewayv2.GetRoutesInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get routes for HTTP API %s: %w", apiID, err)
	}

	var routes []model.APIRoute
	for _, r := range out.Items {
		authType := "NONE"
		if r.AuthorizationType != "" {
			authType = string(r.AuthorizationType)
		}

		routes = append(routes, model.APIRoute{
			RouteKey: aws.ToString(r.RouteKey),
			RouteID:  aws.ToString(r.RouteId),
			Target:   aws.ToString(r.Target),
			AuthType: authType,
		})
	}

	return routes, nil
}
