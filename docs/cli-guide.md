# KaskManager CLI Guide

The KaskManager CLI (`kaskman`) provides comprehensive command-line access to the KaskManager R&D Platform. This guide covers installation, configuration, and usage of all available commands.

## Installation

### Building from Source

1. **Prerequisites**:
   - Go 1.23.0 or later
   - Make (optional, for using Makefile)

2. **Build the CLI**:
   ```bash
   # Using Make
   make build-cli
   
   # Or directly with Go
   go build -o build/kaskman ./cmd/cli
   ```

3. **Install the CLI**:
   ```bash
   # Install to GOPATH/bin
   make install-cli
   
   # Or install to /usr/local/bin (requires sudo)
   make install-local
   
   # Or manually copy
   cp build/kaskman /usr/local/bin/kaskman
   ```

### Cross-Platform Builds

Build for multiple platforms:
```bash
make build-all
```

This creates binaries for Linux, macOS, and Windows in the `build/` directory.

## Quick Start

1. **Check CLI version**:
   ```bash
   kaskman version
   ```

2. **Initialize configuration**:
   ```bash
   kaskman config init
   ```

3. **Authenticate with server**:
   ```bash
   kaskman auth login
   ```

4. **Check system status**:
   ```bash
   kaskman status
   ```

## Global Flags

All commands support these global flags:

- `-u, --url string`: KaskManager server URL (default: http://localhost:8080)
- `-f, --format string`: Output format - table, json, yaml (default: table)
- `-t, --timeout int`: Request timeout in seconds (default: 30)
- `-v, --verbose`: Enable verbose output
- `-h, --help`: Show help information

## Commands Reference

### Authentication (`kaskman auth`)

Manage authentication with the KaskManager server.

#### `kaskman auth login`
Authenticate with username and password.

```bash
# Interactive login
kaskman auth login

# With flags
kaskman auth login -u username -p password
```

**Flags:**
- `-u, --username string`: Username for authentication
- `-p, --password string`: Password for authentication

#### `kaskman auth logout`
Clear stored authentication credentials.

```bash
kaskman auth logout
```

#### `kaskman auth status`
Show current authentication status and user information.

```bash
kaskman auth status
```

### Server Management (`kaskman server`)

Manage the KaskManager server process.

#### `kaskman server start`
Start the KaskManager server.

```bash
# Start in foreground
kaskman server start

# Start as daemon
kaskman server start --daemon

# Custom port and host
kaskman server start --port 9090 --host 0.0.0.0

# With configuration file
kaskman server start --config /path/to/config.yaml
```

**Flags:**
- `-p, --port int`: Port to run the server on (default: 8080)
- `-H, --host string`: Host to bind the server to (default: 0.0.0.0)
- `-d, --daemon`: Run server in daemon mode
- `-c, --config string`: Path to configuration file
- `-l, --log-level string`: Log level (debug, info, warn, error)

#### `kaskman server stop`
Stop a running KaskManager server daemon.

```bash
kaskman server stop
```

#### `kaskman server restart`
Restart the KaskManager server.

```bash
kaskman server restart
```

### System Status (`kaskman status`)

Display comprehensive system status and health information.

```bash
# Basic status
kaskman status

# Detailed status with service information
kaskman status --detailed

# Watch status in real-time
kaskman status --watch --interval 5
```

**Flags:**
- `-d, --detailed`: Show detailed status information
- `-w, --watch`: Watch status in real-time
- `-i, --interval int`: Refresh interval in seconds for watch mode (default: 5)

### Project Management (`kaskman project`)

Manage research and development projects.

#### `kaskman project list`
List all projects with optional filtering.

```bash
# List all projects
kaskman project list

# Filter by status
kaskman project list --status active

# Filter by type and priority
kaskman project list --type research --priority high

# JSON output
kaskman project list --format json
```

**Flags:**
- `-s, --status string`: Filter by status (active, completed, paused, cancelled)
- `-t, --type string`: Filter by type (research, development, analysis, innovation)
- `-p, --priority string`: Filter by priority (low, medium, high, critical)

#### `kaskman project create`
Create a new project.

```bash
# Interactive mode
kaskman project create --interactive

# With flags
kaskman project create \
  --name "AI Research Project" \
  --description "Research into new AI algorithms" \
  --type research \
  --priority high \
  --estimated-hours 160 \
  --budget 50000 \
  --tags "ai,research,algorithms"
```

**Flags:**
- `-n, --name string`: Project name
- `-d, --description string`: Project description
- `-t, --type string`: Project type (research, development, analysis, innovation)
- `-p, --priority string`: Priority (low, medium, high, critical)
- `-e, --estimated-hours int`: Estimated hours
- `-b, --budget float`: Budget
- `-T, --tags strings`: Tags (comma-separated)
- `-i, --interactive`: Interactive mode

#### `kaskman project show <project-id>`
Display detailed information about a specific project.

```bash
kaskman project show 550e8400-e29b-41d4-a716-446655440000
```

#### `kaskman project update <project-id>`
Update project details.

```bash
kaskman project update 550e8400-e29b-41d4-a716-446655440000 \
  --status completed \
  --progress 100 \
  --priority low
```

**Flags:**
- `-n, --name string`: Project name
- `-d, --description string`: Project description
- `-s, --status string`: Status (active, completed, paused, cancelled)
- `-p, --priority string`: Priority (low, medium, high, critical)
- `-P, --progress int`: Progress (0-100)
- `-e, --estimated-hours int`: Estimated hours
- `-b, --budget float`: Budget
- `-T, --tags strings`: Tags (comma-separated)

#### `kaskman project delete <project-id>`
Delete a project (requires confirmation).

```bash
# With confirmation prompt
kaskman project delete 550e8400-e29b-41d4-a716-446655440000

# Force delete without confirmation
kaskman project delete 550e8400-e29b-41d4-a716-446655440000 --force
```

**Flags:**
- `-f, --force`: Force delete without confirmation

### Task Management (`kaskman task`)

Manage tasks within projects.

#### `kaskman task list`
List all tasks with optional filtering.

```bash
# List all tasks
kaskman task list

# Filter by status and type
kaskman task list --status in_progress --type coding

# Filter by project
kaskman task list --project 550e8400-e29b-41d4-a716-446655440000
```

**Flags:**
- `-s, --status string`: Filter by status (pending, in_progress, completed, failed)
- `-t, --type string`: Filter by type (analysis, coding, research, testing, documentation, design)
- `-p, --priority string`: Filter by priority (low, medium, high, critical)
- `-P, --project string`: Filter by project ID

#### `kaskman task create`
Create a new task.

```bash
# Interactive mode
kaskman task create --interactive

# With flags
kaskman task create \
  --title "Implement user authentication" \
  --description "Add JWT-based authentication system" \
  --type coding \
  --priority high \
  --estimated-time 480 \
  --project 550e8400-e29b-41d4-a716-446655440000
```

**Flags:**
- `-t, --title string`: Task title
- `-d, --description string`: Task description
- `-T, --type string`: Task type (analysis, coding, research, testing, documentation, design)
- `-p, --priority string`: Priority (low, medium, high, critical)
- `-e, --estimated-time int`: Estimated time in minutes
- `-P, --project string`: Project ID
- `-u, --assigned-to string`: Assigned user ID
- `-a, --agent string`: Agent ID
- `-i, --interactive`: Interactive mode

#### `kaskman task show <task-id>`
Display detailed information about a specific task.

```bash
kaskman task show 550e8400-e29b-41d4-a716-446655440000
```

#### `kaskman task update <task-id>`
Update task details.

```bash
kaskman task update 550e8400-e29b-41d4-a716-446655440000 \
  --status in_progress \
  --progress 50
```

**Flags:**
- `-t, --title string`: Task title
- `-d, --description string`: Task description
- `-s, --status string`: Status (pending, in_progress, completed, failed)
- `-p, --priority string`: Priority (low, medium, high, critical)
- `-P, --progress int`: Progress (0-100)
- `-e, --estimated-time int`: Estimated time in minutes
- `-r, --result string`: Task result
- `-j, --project string`: Project ID
- `-u, --assigned-to string`: Assigned user ID
- `-a, --agent string`: Agent ID

#### `kaskman task complete <task-id>`
Mark a task as completed.

```bash
# With result
kaskman task complete 550e8400-e29b-41d4-a716-446655440000 \
  --result "Authentication system implemented successfully"

# Interactive
kaskman task complete 550e8400-e29b-41d4-a716-446655440000
```

**Flags:**
- `-r, --result string`: Task result

#### `kaskman task assign <task-id> <user-id-or-agent-id>`
Assign a task to a user or agent.

```bash
# Assign to user
kaskman task assign 550e8400-e29b-41d4-a716-446655440000 user-id-here

# Assign to agent
kaskman task assign 550e8400-e29b-41d4-a716-446655440000 agent-id-here --agent
```

**Flags:**
- `-a, --agent`: Assign to agent instead of user

#### `kaskman task delete <task-id>`
Delete a task (requires confirmation).

```bash
kaskman task delete 550e8400-e29b-41d4-a716-446655440000
```

**Flags:**
- `-f, --force`: Force delete without confirmation

### Agent Management (`kaskman agent`)

Manage AI agents in the platform.

#### `kaskman agent list`
List all agents with optional filtering.

```bash
# List all agents
kaskman agent list

# Filter by status and type
kaskman agent list --status active --type researcher
```

**Flags:**
- `-s, --status string`: Filter by status (active, inactive, busy, error)
- `-t, --type string`: Filter by type (researcher, coder, analyst, tester, designer, coordinator, optimizer)

#### `kaskman agent spawn`
Create and spawn a new AI agent.

```bash
# Interactive mode
kaskman agent spawn --interactive

# With flags
kaskman agent spawn \
  --name "Research Assistant" \
  --type researcher \
  --capabilities "data_analysis,pattern_recognition,report_generation"
```

**Flags:**
- `-n, --name string`: Agent name
- `-t, --type string`: Agent type (researcher, coder, analyst, tester, designer, coordinator, optimizer)
- `-c, --capabilities strings`: Agent capabilities
- `-i, --interactive`: Interactive mode

#### `kaskman agent show <agent-id>`
Display detailed information about a specific agent.

```bash
kaskman agent show 550e8400-e29b-41d4-a716-446655440000
```

#### `kaskman agent update <agent-id>`
Update agent configuration.

```bash
kaskman agent update 550e8400-e29b-41d4-a716-446655440000 \
  --status active \
  --capabilities "data_analysis,machine_learning,visualization"
```

**Flags:**
- `-n, --name string`: Agent name
- `-s, --status string`: Agent status (active, inactive, busy, error)
- `-c, --capabilities strings`: Agent capabilities

#### `kaskman agent start <agent-id>`
Activate an agent to begin processing tasks.

```bash
kaskman agent start 550e8400-e29b-41d4-a716-446655440000
```

#### `kaskman agent stop <agent-id>`
Deactivate an agent to stop processing tasks.

```bash
kaskman agent stop 550e8400-e29b-41d4-a716-446655440000
```

#### `kaskman agent monitor`
Monitor real-time agent activity and performance.

```bash
# Monitor with 5-second refresh
kaskman agent monitor

# Custom refresh interval
kaskman agent monitor --interval 10
```

**Flags:**
- `-i, --interval int`: Refresh interval in seconds (default: 5)

#### `kaskman agent delete <agent-id>`
Delete an agent (requires confirmation).

```bash
kaskman agent delete 550e8400-e29b-41d4-a716-446655440000
```

**Flags:**
- `-f, --force`: Force delete without confirmation

### R&D Operations (`kaskman rnd`)

Perform research and development operations including pattern analysis and project generation.

#### `kaskman rnd analyze`
Trigger pattern analysis to identify trends and insights.

```bash
# Interactive mode
kaskman rnd analyze --interactive

# With specific parameters
kaskman rnd analyze \
  --type system_usage \
  --depth 5 \
  --time-range 24h \
  --context "Performance optimization"
```

**Flags:**
- `-t, --type string`: Pattern type (user_behavior, system_usage, project_trend, performance, error_pattern)
- `-c, --context string`: Analysis context
- `-d, --depth int`: Analysis depth (default: 5)
- `-r, --time-range string`: Time range (1h, 24h, 7d, 30d, all) (default: 24h)
- `-i, --interactive`: Interactive mode

#### `kaskman rnd generate`
Generate project suggestions based on patterns and insights.

```bash
# Interactive mode
kaskman rnd generate --interactive

# With parameters
kaskman rnd generate \
  --category development \
  --focus "machine learning" \
  --priority high \
  --max-projects 10
```

**Flags:**
- `-c, --category string`: Project category (research, development, analysis, innovation, optimization)
- `-f, --focus string`: Focus area
- `-p, --priority string`: Priority (low, medium, high, critical)
- `-m, --max-projects int`: Maximum projects to generate (default: 5)
- `-i, --interactive`: Interactive mode

#### `kaskman rnd insights`
List all generated insights from pattern analysis.

```bash
# List all insights
kaskman rnd insights

# Filter by impact and type
kaskman rnd insights --impact high --type optimization
```

**Flags:**
- `-i, --impact string`: Filter by impact (low, medium, high, critical)
- `-t, --type string`: Filter by type (optimization, recommendation, warning, trend)

#### `kaskman rnd patterns`
List all discovered patterns from analysis.

```bash
# List all patterns
kaskman rnd patterns

# Filter by type and confidence
kaskman rnd patterns --type user_behavior --min-confidence 0.8
```

**Flags:**
- `-t, --type string`: Filter by pattern type
- `-c, --min-confidence float`: Minimum confidence threshold

#### `kaskman rnd coordinate`
Coordinate multiple agents to work on a task or project.

```bash
# Interactive mode
kaskman rnd coordinate --interactive

# Coordinate for a task
kaskman rnd coordinate \
  --task 550e8400-e29b-41d4-a716-446655440000 \
  --agent-types "researcher,coder,tester" \
  --strategy collaborative

# Coordinate for a project
kaskman rnd coordinate \
  --project 550e8400-e29b-41d4-a716-446655440000 \
  --agent-types "researcher,analyst" \
  --strategy parallel
```

**Flags:**
- `-t, --task string`: Task ID
- `-p, --project string`: Project ID
- `-a, --agent-types strings`: Agent types to coordinate
- `-s, --strategy string`: Coordination strategy (sequential, parallel, hierarchical, collaborative)
- `-i, --interactive`: Interactive mode

#### `kaskman rnd stats`
Display comprehensive R&D statistics and performance metrics.

```bash
kaskman rnd stats
```

### Configuration (`kaskman config`)

Manage CLI and server configuration.

#### `kaskman config show`
Display current configuration.

```bash
# Show CLI configuration
kaskman config show

# Show server configuration
kaskman config show --server
```

**Flags:**
- `-s, --server`: Show server configuration

#### `kaskman config set <key> <value>`
Set a configuration value.

```bash
kaskman config set default_url http://localhost:9090
kaskman config set timeout 60
```

#### `kaskman config get <key>`
Get a configuration value.

```bash
kaskman config get default_url
```

#### `kaskman config init`
Initialize CLI configuration with default values.

```bash
# Initialize (fails if config exists)
kaskman config init

# Force initialization
kaskman config init --force
```

**Flags:**
- `-f, --force`: Force initialization, overwrite existing config

#### `kaskman config validate`
Validate configuration.

```bash
# Validate CLI configuration
kaskman config validate

# Validate server configuration
kaskman config validate --server
```

**Flags:**
- `-s, --server`: Validate server configuration

#### `kaskman config edit`
Open configuration file in default editor.

```bash
kaskman config edit
```

### Shell Completion (`kaskman completion`)

Generate and manage shell completion scripts.

#### `kaskman completion <shell>`
Generate completion script for your shell.

```bash
# Bash
kaskman completion bash

# Zsh
kaskman completion zsh

# Fish
kaskman completion fish

# PowerShell
kaskman completion powershell
```

#### `kaskman completion install [shell]`
Install completion script to the appropriate location.

```bash
# Auto-detect shell and install
kaskman completion install

# Install for specific shell
kaskman completion install bash
```

#### `kaskman completion uninstall [shell]`
Remove completion script.

```bash
kaskman completion uninstall bash
```

#### `kaskman completion status`
Show installation status of completion scripts.

```bash
kaskman completion status
```

## Configuration

### CLI Configuration

The CLI uses a YAML configuration file located at `~/.kaskman/config.yaml`. Initialize it with:

```bash
kaskman config init
```

Example configuration:
```yaml
default_url: http://localhost:8080
default_format: table
timeout: 30
retry_attempts: 3
retry_delay: 1s
log_level: info
auto_update_check: true
colors: true
show_headers: true
show_timestamps: true
completion_cache_ttl: 1h
```

### Authentication

Authentication credentials are stored in `~/.kaskman/auth.json`. This file is created automatically when you run `kaskman auth login`.

### Environment Variables

You can override configuration with environment variables:

- `KASKMAN_URL`: Server URL
- `KASKMAN_TOKEN`: Authentication token
- `KASKMAN_FORMAT`: Output format
- `KASKMAN_TIMEOUT`: Request timeout
- `EDITOR`: Editor for `config edit` command

## Output Formats

The CLI supports multiple output formats:

### Table (Default)
Human-readable tabular output with colors and formatting.

### JSON
Machine-readable JSON output for scripting:
```bash
kaskman project list --format json | jq '.[] | .name'
```

### YAML
YAML output for configuration files:
```bash
kaskman project show <id> --format yaml
```

## Error Handling

The CLI provides comprehensive error handling with:

- Clear error messages
- Suggestions for fixing common issues
- Proper exit codes for scripting
- Verbose mode for debugging

## Examples

### Complete Workflow Example

```bash
# 1. Initialize and authenticate
kaskman config init
kaskman auth login

# 2. Create a project
kaskman project create \
  --name "AI Chat Bot" \
  --description "Develop an intelligent chat bot" \
  --type development \
  --priority high \
  --interactive

# 3. Create tasks for the project
PROJECT_ID="<project-id-from-step-2>"
kaskman task create \
  --title "Design conversation flow" \
  --type design \
  --project $PROJECT_ID

kaskman task create \
  --title "Implement NLP processing" \
  --type coding \
  --project $PROJECT_ID

# 4. Spawn agents to work on tasks
kaskman agent spawn \
  --name "Design Agent" \
  --type designer

kaskman agent spawn \
  --name "Coding Agent" \
  --type coder

# 5. Coordinate agents for the project
kaskman rnd coordinate \
  --project $PROJECT_ID \
  --agent-types "designer,coder" \
  --strategy collaborative

# 6. Monitor progress
kaskman project show $PROJECT_ID
kaskman task list --project $PROJECT_ID
kaskman agent monitor
```

### Scripting Example

```bash
#!/bin/bash
# Automated project status report

# Get all active projects
PROJECTS=$(kaskman project list --status active --format json)

# Generate report
echo "# Active Projects Report"
echo "Generated on: $(date)"
echo

echo "$PROJECTS" | jq -r '.[] | "## \(.name)\n- Status: \(.status)\n- Progress: \(.progress)%\n- Priority: \(.priority)\n"'

# Get system status
echo "## System Status"
kaskman status --detailed
```

## Troubleshooting

### Common Issues

1. **Connection refused**:
   - Ensure the KaskManager server is running
   - Check the server URL in configuration
   - Verify network connectivity

2. **Authentication failed**:
   - Check username and password
   - Ensure user account is active
   - Try logging out and back in

3. **Command not found**:
   - Ensure CLI is in your PATH
   - Try reinstalling the CLI
   - Check file permissions

4. **Permission denied**:
   - Check file permissions on config directory
   - Ensure you have write access to installation directory

### Getting Help

- Use `--help` flag with any command
- Check verbose output with `--verbose`
- Review server logs for API-related issues
- Check the project documentation

## Development

### Building from Source

```bash
# Clone the repository
git clone <repository-url>
cd KaskMan

# Install dependencies
npm install

# Build the CLI
make build-cli

# Run tests
make test

# Install locally
make install-cli
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

For more information, see the project's contributing guidelines.