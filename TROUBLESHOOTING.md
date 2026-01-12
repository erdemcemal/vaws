# Troubleshooting

## Prerequisites

### Required Tools

| Tool | Purpose | Installation |
|------|---------|--------------|
| AWS CLI v2 | AWS authentication | [Install Guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) |
| Session Manager Plugin | Port forwarding via SSM | `brew install --cask session-manager-plugin` |

### AWS Configuration

```bash
# Configure credentials (choose one)
aws configure                          # Access keys
aws configure sso                      # SSO (recommended)

# Login if using SSO
aws sso login --profile your-profile
```

### IAM Permissions

Your IAM role needs these permissions:

```
cloudformation:DescribeStacks, cloudformation:ListStackResources
ecs:ListClusters, ecs:ListServices, ecs:DescribeServices, ecs:ListTasks, ecs:DescribeTasks
lambda:ListFunctions, lambda:GetFunction, lambda:InvokeFunction
apigateway:GET
apigatewayv2:GetApis, apigatewayv2:GetStages, apigatewayv2:GetRoutes
sqs:ListQueues, sqs:GetQueueAttributes
dynamodb:ListTables, dynamodb:DescribeTable, dynamodb:Query, dynamodb:Scan
ec2:DescribeInstances, ec2:DescribeVpcEndpoints
ssm:StartSession, ssm:DescribeInstanceInformation
logs:FilterLogEvents, logs:GetLogEvents
```

---

## Common Issues

### ECS Port Forwarding: "TargetNotConnected"

```
TargetNotConnected: ecs:ecs-service-name_xxx is not connected
```

**Cause:** The ECS task doesn't have SSM connectivity.

**Solutions:**

1. **Enable Execute Command on the service:**
   ```bash
   aws ecs update-service --cluster <cluster> --service <service> --enable-execute-command
   ```

2. **Add SSM permissions to task role:**
   ```json
   {
     "Effect": "Allow",
     "Action": [
       "ssmmessages:CreateControlChannel",
       "ssmmessages:CreateDataChannel",
       "ssmmessages:OpenControlChannel",
       "ssmmessages:OpenDataChannel"
     ],
     "Resource": "*"
   }
   ```

3. **For private subnets, add VPC endpoints:**
   - `com.amazonaws.<region>.ssmmessages`
   - `com.amazonaws.<region>.ssm`

4. **Redeploy tasks** after enabling (existing tasks won't have SSM agent):
   ```bash
   aws ecs update-service --cluster <cluster> --service <service> --force-new-deployment
   ```

### Private API Gateway: "No VPC endpoint configured"

**Cause:** The API Gateway doesn't have a VPC endpoint ID in its configuration.

**Solutions:**

1. Check that the API Gateway is configured with `endpointConfiguration.vpcEndpointIds`
2. Or add `vpc_endpoint_id` to `~/.vaws/config.yaml` for cross-account access

### Jump Host: "No suitable jump host found"

**Cause:** No SSM-managed EC2 instances found.

**Solutions:**

1. Ensure EC2 instance has SSM agent running and is online
2. Check instance IAM role has `AmazonSSMManagedInstanceCore` policy
3. Configure a specific jump host in `~/.vaws/config.yaml`:
   ```yaml
   profiles:
     your-profile:
       jump_host: your-instance-name
   ```

---

## Port Forwarding Details

### ECS Services (Fargate/EC2)

Forward traffic to containers running in ECS tasks.

**Requirements:**
- ECS task with `enableExecuteCommand: true`
- Task role with SSM permissions

**Steps:**
1. Navigate: **Stack → Services → Select service**
2. Press `p`, enter local port (or Enter for random)
3. Access at `http://localhost:<port>`

### Private API Gateway (Lambda Backend)

Access private API Gateways that invoke Lambda functions within a VPC. This is how you reach Lambda functions that aren't publicly accessible.

**Requirements:**
- Private API Gateway with VPC endpoint
- EC2 instance in the same VPC (jump host) with SSM agent
- Jump host's security group must allow outbound HTTPS (443)

**Architecture:**
```
localhost → SSM Tunnel → EC2 Jump Host → VPC Endpoint → API Gateway → Lambda
```

**Steps:**
1. Navigate: **API Gateway → Select private API → Stages → Select stage**
2. Press `p`, enter local port
3. Select a jump host from the list (SSM-managed EC2 instances)
4. Access at `http://localhost:<port>/your-endpoint`

**How it works:**
- Creates an SSM tunnel through the jump host to the API Gateway VPC endpoint
- Runs a local HTTP proxy that handles TLS termination
- Automatically prepends the stage name to your requests
- No need to modify `/etc/hosts` or use `curl --resolve`

**Example:**
```bash
# Your request
curl http://localhost:8080/users/123

# Gets forwarded as
https://<api-id>-<vpce-id>.execute-api.<region>.amazonaws.com/prod/users/123
```

---

## Configuration Reference

### Config File Location

`~/.vaws/config.yaml`

### Full Configuration Example

```yaml
profiles:
  production:
    jump_host: bastion-prod      # Preferred jump host name or instance ID
    region: us-east-1
  staging:
    jump_host: bastion-staging
    vpc_endpoint_id: vpce-xxx    # For cross-account API Gateway access

defaults:
  jump_host_tags:                # Auto-discovery by tags
    - "vaws:jump-host=true"
    - "Name=bastion"
  jump_host_names:               # Auto-discovery by name
    - "bastion"
    - "jumphost"
```

### Data Storage

| File | Purpose |
|------|---------|
| `~/.vaws/config.yaml` | User configuration |
| `~/.vaws/tunnels.json` | Persistent tunnel state |

---

## Getting Help

- **GitHub Issues:** [Report a bug](https://github.com/erdemcemal/vaws/issues)
- **Discussions:** [Ask a question](https://github.com/erdemcemal/vaws/discussions)
