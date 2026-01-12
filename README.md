# vaws

**The AWS Console in your terminal.** Navigate CloudFormation, ECS, Lambda, API Gateway, SQS, and DynamoDB with vim-style keybindings.

[![Release](https://img.shields.io/github/v/release/erdemcemal/vaws?style=flat-square)](https://github.com/erdemcemal/vaws/releases)
[![Go](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)

![vaws demo](assets/demo.gif)

## Why vaws?

- **20 clicks → 2 keystrokes** - Jump from CloudFormation stack to running ECS task logs instantly
- **Port forward without the AWS CLI dance** - Tunnel to ECS containers and private API Gateways in seconds
- **Multi-account workflow** - Switch profiles and regions on the fly without restarting
- **Keyboard-first** - If you love vim, k9s, or lazygit, you'll feel right at home

## Installation

### Homebrew (macOS/Linux)

```bash
brew install erdemcemal/tap/vaws
```

### Binary Download

Download from [GitHub Releases](https://github.com/erdemcemal/vaws/releases).

### From Source

```bash
git clone https://github.com/erdemcemal/vaws.git
cd vaws && make install
```

> **Note:** Requires AWS CLI v2 configured (`aws configure` or `aws sso login`). For port forwarding, install the [Session Manager Plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html).

## Quick Start

```bash
# Launch with default profile
vaws

# Use a specific profile and region
vaws --profile production --region eu-west-1

# Test AWS connectivity
vaws --test
```

Press `:` to open the command palette or check the shortcuts below.

## Features

| Service | What You Can Do |
|---------|-----------------|
| **CloudFormation** | Browse stacks, outputs, parameters, and resources |
| **ECS** | View services, tasks, deployments, and stream CloudWatch logs |
| **Lambda** | List functions, view details, invoke with custom payloads |
| **API Gateway** | Explore REST/HTTP APIs, stages, and routes |
| **SQS** | Browse queues with DLQ visibility and message counts |
| **DynamoDB** | Query and scan tables with paginated results |
| **Port Forwarding** | Tunnel to ECS containers and private API Gateways via SSM |

## Real-World Workflows

### Debug a Failing ECS Service

```
1. Launch vaws
2. Press 2 to view ECS services
3. Navigate to your service with j/k
4. Press l to stream CloudWatch logs
5. Press p to port forward and test locally
```

### Access a Private API Gateway

```
1. Press 4 to view API Gateways
2. Select your private API → Stage
3. Press p, pick a jump host
4. curl http://localhost:8080/your-endpoint
```

### Browse DynamoDB Tables

```
1. Press 6 to view DynamoDB tables
2. Select a table, press Enter
3. Press q to query or s to scan
4. Navigate results with j/k, paginate with n/p
```

## Keyboard Shortcuts

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Select / Drill down |
| `Esc` | Go back |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `/` | Filter current list |

### Views

| Key | Action |
|-----|--------|
| `1` | CloudFormation Stacks |
| `2` | ECS Services |
| `3` | Lambda Functions |
| `4` | API Gateway |
| `5` | Active Tunnels |
| `6` | DynamoDB Tables |
| `:` | Command palette |

### Actions

| Key | Action |
|-----|--------|
| `p` | Port forward |
| `r` | Refresh |
| `l` | Toggle logs |
| `t` | View tunnels |
| `x` | Stop tunnel |
| `c` | Clear terminated |
| `q` | Quit |

## Configuration (Optional)

Create `~/.vaws/config.yaml` for advanced setups:

```yaml
profiles:
  production:
    jump_host: bastion-prod   # Preferred SSM jump host
    region: us-east-1

defaults:
  jump_host_tags:
    - "vaws:jump-host=true"
```

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for detailed configuration options and common issues.

## Roadmap

- [ ] Lambda invocation with payload editor
- [ ] SQS message send/peek/redrive
- [ ] Secrets Manager browser
- [ ] Global search across all resources
- [ ] CloudWatch alarms dashboard

See [ROADMAP.md](ROADMAP.md) for the full plan.

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) to get started.

**Found a bug?** [Open an issue](https://github.com/erdemcemal/vaws/issues)

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<p align="center">
  <sub>Built for engineers who'd rather type than click.</sub><br>
  <sub>If vaws saves you time, consider giving it a <a href="https://github.com/erdemcemal/vaws">star</a></sub>
</p>
