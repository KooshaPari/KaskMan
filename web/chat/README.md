# KaskMan Chat Interface

A Claude Desktop-style chat interface for project management.

## Quick Start

1. **Start the server:**
   ```bash
   cd cmd/chat-server
   go run main.go
   ```

2. **Open the chat interface:**
   - Navigate to `http://localhost:8080/` in your browser
   - The interface will automatically load with sample project data

## Features

- Claude Desktop-style chat interface
- Project overview and status tracking
- TUI component parsing and display  
- Real-time WebSocket communication
- Interactive project navigation

## API Endpoints

- `GET /` - Chat interface
- `GET /api/v1/health` - Health check
- `GET /api/v1/projects` - List projects
- `GET /api/v1/projects/:id` - Get project details
- `POST /api/v1/sessions` - Create chat session
- `POST /api/v1/chat/message` - Send message
- `WS /ws` - WebSocket connection

## Development

The chat server runs on port 8080 by default. Use `--port` flag to change.

For development mode with detailed logging:
```bash
go run main.go --dev --log-level debug
```