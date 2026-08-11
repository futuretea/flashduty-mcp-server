# FlashDuty MCP Server

[![Build](https://github.com/futuretea/flashduty-mcp-server/actions/workflows/build.yaml/badge.svg)](https://github.com/futuretea/flashduty-mcp-server/actions/workflows/build.yaml)
[![GitHub License](https://img.shields.io/github/license/futuretea/flashduty-mcp-server)](https://github.com/futuretea/flashduty-mcp-server/blob/main/LICENSE)
[![npm](https://img.shields.io/npm/v/@futuretea/flashduty-mcp-server)](https://www.npmjs.com/package/@futuretea/flashduty-mcp-server)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/futuretea/flashduty-mcp-server?sort=semver)](https://github.com/futuretea/flashduty-mcp-server/releases/latest)

[Features](#features) | [Getting Started](#getting-started) | [Configuration](#configuration) | [Tools](#tools) | [Development](#development)

## Features <a id="features"></a>

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for [FlashDuty](https://flashcat.cloud/) incident management.

- **Incident Management**: List, create, acknowledge, resolve, reopen, snooze, and comment on incidents
- **Alert Management**: List and get alert details
- **Collaboration Spaces**: List and inspect channels (collaboration spaces)
- **Team & Member Lookup**: List teams and members
- **On-call Schedules**: Query on-call schedules by team
- **Incident Statistics**: MTTA, MTTR, counts, ack rate, and noise reduction metrics
- **Incident Timeline**: View event feed (comments, acks, resolves, escalations, notifications)
- **Incident-Alert Association**: See which alerts belong to a given incident
- **Brief Mode**: Reduce response size for LLM token limits by returning only key fields
- **Advanced Filtering**: Filter by channel, responder, acknowledger, creator, and severity
- **Security Controls**: `read_only` mode to disable all write operations
- **Dual Transport**: Stdio mode for MCP client integration or HTTP/SSE mode for network access
- **Cross-platform**: Native binaries for Linux, macOS, Windows (amd64/arm64), npm package, and Docker images

## Getting Started <a id="getting-started"></a>

### Requirements

- A FlashDuty API app key (from [FlashDuty Console](https://console.flashcat.cloud/) -> Personal Settings -> API Keys)

### Claude Code

```shell
claude mcp add flashduty -- npx @futuretea/flashduty-mcp-server@latest \
  --app-key YOUR_APP_KEY
```

### VS Code / Cursor

Add to `.vscode/mcp.json` or `~/.cursor/mcp.json`:

```json
{
  "servers": {
    "flashduty": {
      "command": "npx",
      "args": [
        "-y",
        "@futuretea/flashduty-mcp-server@latest",
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
npx @futuretea/flashduty-mcp-server@latest --help
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
| `--insecure-skip-tls-verify` | Skip FlashDuty API TLS certificate verification | `false` |
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

# Only for trusted internal FlashDuty API endpoints with a private or self-signed CA.
# insecure_skip_tls_verify: false

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
FLASHDUTY_MCP_INSECURE_SKIP_TLS_VERIFY=false
```

For trusted internal FlashDuty API endpoints with a private or self-signed CA, set
`FLASHDUTY_MCP_INSECURE_SKIP_TLS_VERIFY=true` or pass
`--insecure-skip-tls-verify`. Keep this disabled for public endpoints. Prefer
installing the private CA in the runtime trust store when possible.

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

Use `--read-only` to disable all write operations (create, acknowledge, resolve, reopen, snooze, comment, update, assign).

### Incident Tools

<details>
<summary>list_incidents</summary>

List incidents from FlashDuty. Use `brief=true` to reduce response size (recommended for initial queries).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `time_range` | string | No | Relative time range up to 31 days: `1h`, `24h`, `7d`, `30d`, `1w`, `last_day`, `last_week`, or `week_before_last`. Alternative to `start_time`+`end_time`. |
| `start_time` | integer | Conditional | Search interval start time in unix seconds. Required if `time_range` is not set. |
| `end_time` | integer | Conditional | Search interval end time in unix seconds. Required if `time_range` is not set. |
| `brief` | boolean | No | If true, return only key fields to reduce data volume. |
| `progress` | string | No | Filter by progress: `Triggered`, `Processing`, `Closed`, or comma-separated. |
| `incident_severity` | string | No | Filter by severity: `Critical`, `Warning`, `Info`, or comma-separated. |
| `query` | string | No | Full-text search keyword for incidents. |
| `channel_ids` | integer[] | No | Filter by collaboration space IDs. |
| `responder_ids` | integer[] | No | Filter by responder person IDs. |
| `acker_ids` | integer[] | No | Filter by acknowledger person IDs. |
| `creator_ids` | integer[] | No | Filter by creator person IDs. `0` = system-generated. |
| `limit` | integer | No | Page size (1-100, default 20). |
| `p` | integer | No | Page number starting from 1. |

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
| `title` | string | No | Optional incident title. |
| `incident_severity` | string | Yes | Severity: `Critical`, `Warning`, or `Info`. |
| `description` | string | No | Incident description (plain text or markdown). |
| `channel_id` | integer | No | Collaboration space ID. |

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
| `reason` | string | No | Optional reason for reopening. |

</details>

<details>
<summary>snooze_incidents</summary>

Snooze (temporarily mute) one or more incidents. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_ids` | string[] | Yes | List of incident IDs to snooze. |
| `minutes` | integer | Yes | Minutes to snooze (1-1440). |

</details>

<details>
<summary>comment_incidents</summary>

Add a comment to one or more incidents. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_ids` | string[] | Yes | List of incident IDs to comment on. |
| `comment` | string | No | Optional comment content (1-1024 characters when provided). |

</details>

<details>
<summary>update_incident</summary>

Update an incident's details. At least one field besides `incident_id` must be provided. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_id` | string | Yes | The incident ID to update. |
| `title` | string | No | New incident title. |
| `description` | string | No | New incident description. Supports markdown. |
| `incident_severity` | string | No | New severity: `Critical`, `Warning`, or `Info`. |
| `impact` | string | No | Impact description of the incident. |
| `root_cause` | string | No | Root cause of the incident. |
| `resolution` | string | No | Resolution description of the incident. |

</details>

<details>
<summary>assign_incident</summary>

Assign one or more incidents to specific persons or an escalation rule. Disabled when `read_only=true`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_ids` | string[] | Yes | List of incident IDs to assign. |
| `type` | string | No | Assignment type: `assign`, `reassign`, `escalate`, or `reopen`. Default: `assign`. |
| `person_ids` | integer[] | No | List of person IDs to assign to (when type is `assign`). |
| `escalate_rule_id` | string | No | Escalation rule ObjectID (when type is `escalate`). Provide this or `person_ids`. |

</details>

### Alert Tools

<details>
<summary>list_alerts</summary>

List alerts from FlashDuty. Use `brief=true` to reduce response size.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `time_range` | string | No | Relative time range up to 31 days: `1h`, `24h`, `7d`, `30d`, `1w`, `last_day`, `last_week`, or `week_before_last`. Alternative to `start_time`+`end_time`. |
| `start_time` | integer | Conditional | Search interval start time in unix seconds. Required if `time_range` is not set. |
| `end_time` | integer | Conditional | Search interval end time in unix seconds. Required if `time_range` is not set. |
| `brief` | boolean | No | If true, return only key fields to reduce data volume. |
| `alert_severity` | string | No | Filter by severity: `Critical`, `Warning`, `Info`, `Ok`, or comma-separated. |
| `is_active` | boolean | No | Filter by active status. |
| `channel_ids` | integer[] | No | Filter by collaboration space IDs. |
| `limit` | integer | No | Page size (1-100, default 20). |
| `p` | integer | No | Page number starting from 1. |

</details>

<details>
<summary>get_alert</summary>

Get detailed information about a specific alert.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `alert_id` | string | Yes | The alert ID. |

</details>

### Channel Tools

<details>
<summary>list_channels</summary>

List collaboration spaces (channels).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | No | Search keyword to match channel name and description. |
| `limit` | integer | No | Page size. |
| `p` | integer | No | Page number starting from 1. |

</details>

<details>
<summary>get_channel</summary>

Get detailed information about a specific channel.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `channel_id` | integer | Yes | The channel ID. |

</details>

### Team & Member Tools

<details>
<summary>list_teams</summary>

List teams from FlashDuty.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | No | Search keyword to match team name and description. |
| `limit` | integer | No | Page size. |
| `p` | integer | No | Page number starting from 1. |

</details>

<details>
<summary>list_members</summary>

List members from FlashDuty.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | No | Search keyword to match name and email. Phone number triggers strict match. |
| `limit` | integer | No | Page size. |
| `p` | integer | No | Page number starting from 1. |

</details>

### Schedule Tools

<details>
<summary>list_schedules</summary>

List on-call schedules.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `team_ids` | integer[] | No | Optional list of team IDs to filter schedules. |
| `query` | string | No | Search keyword to match schedule name. |
| `limit` | integer | No | Page size. |
| `p` | integer | No | Page number starting from 1. |

</details>

### Insight Tools

<details>
<summary>get_incident_stats</summary>

Get aggregated incident statistics (MTTA, MTTR, counts, ack rate, noise reduction).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `time_range` | string | No | Relative time range: `1h`, `24h`, `7d`, `30d`, `1w`, `6M`, or `last_day`, `last_week`, `week_before_last`. Alternative to `start_time`+`end_time`. |
| `start_time` | integer | Conditional | Start time in unix seconds. Required if `time_range` is not set. |
| `end_time` | integer | Conditional | End time in unix seconds. The API allows a maximum one-year span. Required if `time_range` is not set. |
| `channel_ids` | integer[] | No | Filter by collaboration space IDs. |
| `team_ids` | integer[] | No | Filter by team IDs. |
| `severities` | string[] | No | Filter by severities: `Critical`, `Warning`, `Info`, `Ok`. |
| `aggregate_unit` | string | No | Time aggregation: `day`, `week`, or `month`. |
| `query` | string | No | Full-text search keyword for incidents. |
| `labels` | object | No | Filter by labels as key-value pairs. |

</details>

<details>
<summary>get_incident_timeline</summary>

Get the timeline (feed) of an incident.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_id` | string | Yes | The incident ID. |
| `types` | string[] | No | Filter by event types (e.g., `i_comm`, `i_ack`, `i_rslv`, `i_notify`). |
| `limit` | integer | No | Page size. |
| `p` | integer | No | Page number starting from 1. |

</details>

<details>
<summary>list_incident_alerts</summary>

List alerts associated with a specific incident.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_id` | string | Yes | The incident ID. |
| `is_active` | boolean | No | Filter by active status. |
| `limit` | integer | No | Page size (max 1000, default 1000). |
| `p` | integer | No | Page number starting from 1. |

</details>

### Similar Incidents Tools

<details>
<summary>list_similar_incidents</summary>

Find similar historical incidents for a given incident. Useful for pattern analysis and diagnosis.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `incident_id` | string | Yes | The incident ID to find similar incidents for. |
| `limit` | integer | No | Maximum number of similar incidents to return (max 20). |

</details>

### Change Tools

<details>
<summary>query_changes</summary>

Query deployment/change events from FlashDuty. Useful for correlating incidents with recent deployments.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `time_range` | string | No | Relative time range: `1h`, `24h`, `7d`, `30d`, `1w`, `6M`, or `last_day`, `last_week`, `week_before_last`. Alternative to `start_time`+`end_time`. |
| `start_time` | integer | No | Start time in unix seconds. |
| `end_time` | integer | No | End time in unix seconds. |
| `query` | string | No | Search keyword to filter changes. |
| `channel_ids` | integer[] | No | Filter by collaboration space IDs. |
| `integration_ids` | integer[] | No | Filter by integration IDs. |
| `order_by` | string | No | Field to order by (e.g., `start_time`). |
| `asc` | boolean | No | Sort ascending if true, descending if false. |
| `include_events` | boolean | No | Include change events in the response. |
| `limit` | integer | No | Page size (1-100, default 20). |
| `p` | integer | No | Page number starting from 1. |

</details>

### Escalation Tools

<details>
<summary>query_escalation_rules</summary>

Query escalation rules for a collaboration space. Returns the escalation policy configuration.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `channel_id` | integer | Yes | The collaboration space (channel) ID. |

</details>

### Custom Field Tools

<details>
<summary>query_fields</summary>

Query custom field definitions from FlashDuty. Returns all configured custom fields for incidents. No parameters required.

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
npx @modelcontextprotocol/inspector@latest -- npx @futuretea/flashduty-mcp-server@latest
```

## Contributing

Contributions are welcome! Please open an issue or pull request on [GitHub](https://github.com/futuretea/flashduty-mcp-server).

## License

[Apache-2.0](LICENSE)
