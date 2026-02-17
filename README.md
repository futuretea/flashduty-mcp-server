# FlashDuty MCP Server

[![Build](https://github.com/futuretea/flashduty-mcp-server/actions/workflows/build.yaml/badge.svg)](https://github.com/futuretea/flashduty-mcp-server/actions/workflows/build.yaml)
[![GitHub License](https://img.shields.io/github/license/futuretea/flashduty-mcp-server)](https://github.com/futuretea/flashduty-mcp-server/blob/main/LICENSE)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/futuretea/flashduty-mcp-server?sort=semver)](https://github.com/futuretea/flashduty-mcp-server/releases/latest)

[Features](#features) | [Getting Started](#getting-started) | [Configuration](#configuration) | [Tools](#tools) | [Development](#development)

## Features <a id="features"></a>

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for [FlashDuty](https://flashcat.cloud/) incident management.

- **Incident Management**: List, create, acknowledge, resolve, reopen, snooze, and comment on incidents
- **Alert Management**: List, get details, and close alerts
- **Collaboration Spaces**: List and inspect channels (collaboration spaces)
- **Team & Member Lookup**: List teams and members
- **On-call Schedules**: Query on-call schedules by team
- **Incident Statistics**: Aggregated MTTA, MTTR, counts, ack rate, and noise reduction metrics
- **Incident Timeline**: View event feed (comments, acks, resolves, escalations, notifications)
- **Incident-Alert Association**: List alerts associated with a specific incident
- **Brief Mode**: Reduce response size for LLM token limits by returning only key fields
- **Advanced Filtering**: Filter by channel, responder, acknowledger, creator, labels, severity, and more
- **Security Controls**: `read_only` mode to disable all write operations
- **Dual Transport**: Stdio mode for MCP client integration or HTTP/SSE mode for network access
- **Cross-platform**: Native binaries for Linux (amd64/arm64), macOS (amd64/arm64), and Docker images

## Getting Started <a id="getting-started"></a>

### Requirements

- A FlashDuty API app key (from [FlashDuty Console](https://console.flashcat.cloud/) -> Personal Settings -> API Keys)

### Claude Code

```shell
claude mcp add flashduty -- \
  /path/to/flashduty-mcp-server \
  --app-key YOUR_APP_KEY
```

### VS Code / Cursor

Add to `.vscode/mcp.json` or `~/.cursor/mcp.json`:

```json
{
  "servers": {
    "flashduty": {
      "command": "/path/to/flashduty-mcp-server",
      "args": [
        "--app-key",
        "YOUR_APP_KEY"
      ]
    }
  }
}
```

### Docker

Stdio mode:

```shell
docker run --rm -i ghcr.io/futuretea/flashduty-mcp-server:latest \
  --app-key YOUR_APP_KEY
```

HTTP/SSE mode:

```shell
docker run --rm -p 8080:8080 ghcr.io/futuretea/flashduty-mcp-server:latest \
  --port 8080 --app-key YOUR_APP_KEY
```

## Configuration <a id="configuration"></a>

Configuration can be set via CLI flags, environment variables, or a config file.

**Priority (highest to lowest):**
1. Command-line flags
2. Environment variables (prefix: `FLASHDUTY_MCP_`)
3. Configuration file
4. Default values

### CLI Options

```shell
flashduty-mcp-server --help
```

| Option | Description | Default |
|--------|-------------|---------|
| `--config` | Config file path (YAML) | |
| `--port` | Port for HTTP/SSE mode (0 = stdio mode) | `0` |
| `--sse-base-url` | Public base URL for SSE endpoint | |
| `--log-level` | Log level (0-9) | `5` |
| `--app-key` | FlashDuty API app key (**required**) | |
| `--base-url` | FlashDuty API base URL | `https://api.flashcat.cloud` |
| `--read-only` | Disable write operations | `false` |
| `--enabled-tools` | Specific tools to enable | |
| `--disabled-tools` | Specific tools to disable | |

### Configuration File

Create `config.yaml`:

```yaml
port: 0  # 0 for stdio, or set a port like 8080 for HTTP/SSE

log_level: 5

# Get the app_key from your FlashDuty account:
# https://console.flashcat.cloud -> Personal Settings -> API Keys
app_key: your-app-key-here

# base_url: https://api.flashcat.cloud

read_only: false  # Set to true to disable write operations

# enabled_tools: []
# disabled_tools: []
```

### Environment Variables

Use `FLASHDUTY_MCP_` prefix with underscores:

```shell
FLASHDUTY_MCP_PORT=8080
FLASHDUTY_MCP_APP_KEY=your-key
FLASHDUTY_MCP_READ_ONLY=false
FLASHDUTY_MCP_LOG_LEVEL=5
```

### HTTP/SSE Mode

Run with a port number for network access:

```shell
flashduty-mcp-server --port 8080 --app-key YOUR_APP_KEY
```

Endpoints:
- `/healthz` - Health check
- `/mcp` - Streamable HTTP endpoint
- `/sse` - Server-Sent Events endpoint
- `/message` - Message endpoint for SSE clients

With a public URL behind a proxy:

```shell
flashduty-mcp-server --port 8080 \
  --sse-base-url https://your-domain.com:8080 \
  --app-key YOUR_APP_KEY
```

## Tools <a id="tools"></a>

Use `--enabled-tools` / `--disabled-tools` for fine-grained control.

Use `--read-only` to disable all write operations (create, acknowledge, resolve, reopen, snooze, comment, close).

### Incident Tools

<details>
<summary>list_incidents</summary>

List incidents from FlashDuty. Use `brief=true` to reduce response size (recommended for initial queries).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start_time` | number | Yes | Search interval start time in unix seconds. |
| `end_time` | number | Yes | Search interval end time in unix seconds. |
| `brief` | boolean | No | If true, return only key fields to reduce data volume. |
| `progress` | string | No | Filter by progress: `Triggered`, `Processing`, `Closed`, or comma-separated. |
| `incident_severity` | string | No | Filter by severity: `Critical`, `Warning`, `Info`, or comma-separated. |
| `title` | string | No | Filter by title. Supports exact, regex (`/pattern/`), and wildcards (`*`, `?`). |
| `channel_ids` | integer[] | No | Filter by collaboration space IDs. |
| `responder_ids` | integer[] | No | Filter by responder person IDs. |
| `acker_ids` | integer[] | No | Filter by acknowledger person IDs. |
| `creator_ids` | integer[] | No | Filter by creator person IDs. `0` = system-generated. |
| `labels` | object | No | Filter by labels (key -> array of values). |
| `limit` | number | No | Page size (1-100, default 20). |
| `p` | number | No | Page number starting from 1. |

</details>

<details>
<summary>get_incident</summary>

Get detailed information about a specific incident.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_id` | string | Yes | The incident ID. |

</details>

<details>
<summary>create_incident</summary>

Create a new incident in FlashDuty. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `title` | string | Yes | Incident title. |
| `incident_severity` | string | Yes | Severity: `Critical`, `Warning`, or `Info`. |
| `description` | string | No | Incident description (plain text or markdown). |
| `channel_id` | number | No | Collaboration space ID. |

</details>

<details>
<summary>ack_incidents</summary>

Acknowledge (claim) one or more incidents. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_ids` | string[] | Yes | List of incident IDs to acknowledge. |

</details>

<details>
<summary>resolve_incidents</summary>

Resolve (close) one or more incidents. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_ids` | string[] | Yes | List of incident IDs to resolve. |
| `root_cause` | string | No | Root cause of the incident. |
| `resolution` | string | No | How the incident was resolved. |

</details>

<details>
<summary>reopen_incidents</summary>

Reopen one or more previously resolved incidents. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_ids` | string[] | Yes | List of incident IDs to reopen. |
| `reason` | string | Yes | Reason for reopening. |

</details>

<details>
<summary>snooze_incidents</summary>

Snooze (temporarily mute) one or more incidents. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_ids` | string[] | Yes | List of incident IDs to snooze. |
| `minutes` | number | Yes | Minutes to snooze (1-1440). |

</details>

<details>
<summary>comment_incidents</summary>

Add a comment to one or more incidents. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_ids` | string[] | Yes | List of incident IDs to comment on. |
| `comment` | string | Yes | Comment content (1-500 characters). |

</details>

### Alert Tools

<details>
<summary>list_alerts</summary>

List alerts from FlashDuty. Use `brief=true` to reduce response size.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start_time` | number | Yes | Search interval start time in unix seconds. |
| `end_time` | number | Yes | Search interval end time in unix seconds. |
| `brief` | boolean | No | If true, return only key fields to reduce data volume. |
| `alert_severity` | string | No | Filter by severity: `Critical`, `Warning`, `Info`, or comma-separated. |
| `title` | string | No | Filter by title. Supports exact, regex, and wildcards. |
| `is_active` | boolean | No | Filter by active status. |
| `channel_ids` | integer[] | No | Filter by collaboration space IDs. |
| `labels` | object | No | Filter by labels (key -> array of values). |
| `limit` | number | No | Page size (1-100, default 20). |
| `p` | number | No | Page number starting from 1. |

</details>

<details>
<summary>get_alert</summary>

Get detailed information about a specific alert.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `alert_id` | string | Yes | The alert ID. |

</details>

<details>
<summary>close_alerts</summary>

Close one or more alerts. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `alert_ids` | string[] | Yes | List of alert IDs to close. |

</details>

### Channel Tools

<details>
<summary>list_channels</summary>

List collaboration spaces (channels).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | No | Search keyword to match channel name and description. |
| `limit` | number | No | Page size. |
| `p` | number | No | Page number starting from 1. |

</details>

<details>
<summary>get_channel</summary>

Get detailed information about a specific channel.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `channel_id` | number | Yes | The channel ID. |

</details>

### Team & Member Tools

<details>
<summary>list_teams</summary>

List teams from FlashDuty.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | No | Search keyword to match team name and description. |
| `limit` | number | No | Page size. |
| `p` | number | No | Page number starting from 1. |

</details>

<details>
<summary>list_members</summary>

List members from FlashDuty.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | No | Search keyword to match name and email. Phone number triggers strict match. |
| `limit` | number | No | Page size. |
| `p` | number | No | Page number starting from 1. |

</details>

### Schedule Tools

<details>
<summary>list_schedules</summary>

List on-call schedules.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `team_ids` | integer[] | Yes | List of team IDs to filter schedules. |
| `query` | string | No | Search keyword to match schedule name. |
| `limit` | number | No | Page size. |
| `p` | number | No | Page number starting from 1. |

</details>

### Insight Tools

<details>
<summary>get_incident_stats</summary>

Get aggregated incident statistics (MTTA, MTTR, counts, ack rate, noise reduction).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start_time` | number | Yes | Start time in unix seconds. |
| `end_time` | number | Yes | End time in unix seconds (max span 6 months). |
| `channel_ids` | integer[] | No | Filter by collaboration space IDs. |
| `team_ids` | integer[] | No | Filter by team IDs. |
| `severities` | string[] | No | Filter by severities: `Critical`, `Warning`, `Info`. |
| `aggregate_unit` | string | No | Time aggregation: `day`, `week`, or `month`. |
| `query` | string | No | Filter by incident title (fuzzy match). |
| `labels` | object | No | Filter by labels as key-value pairs. |

</details>

<details>
<summary>get_incident_timeline</summary>

Get the timeline (feed) of an incident.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_id` | string | Yes | The incident ID. |
| `types` | string[] | No | Filter by event types (e.g., `i_comm`, `i_ack`, `i_rslv`, `i_notify`). |
| `limit` | number | No | Page size. |
| `p` | number | No | Page number starting from 1. |

</details>

<details>
<summary>list_incident_alerts</summary>

List alerts associated with a specific incident.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_id` | string | Yes | The incident ID. |
| `is_active` | boolean | No | Filter by active status. |
| `limit` | number | No | Page size (max 1000, default 1000). |
| `p` | number | No | Page number starting from 1. |

</details>

## Development <a id="development"></a>

### Build

```shell
make build
```

### Test

```shell
go test ./...
```

### Lint

```shell
make lint
```

### Run with mcp-inspector

```shell
npx @modelcontextprotocol/inspector@latest $(pwd)/flashduty-mcp-server
```

## Contributing

Contributions are welcome! Please open an issue or pull request on [GitHub](https://github.com/futuretea/flashduty-mcp-server).

## License

[Apache-2.0](LICENSE)
