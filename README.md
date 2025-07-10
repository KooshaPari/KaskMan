# KaskManager R&D Platform

A high-performance Research & Development management platform built in **Go**, designed for continuous research, development, and intelligent project management.

## 🚀 Features

- **High-Performance Go Backend**: Built with Gin framework for maximum throughput
- **Self-Learning R&D Module**: AI-driven pattern recognition and project generation
- **Real-Time WebSockets**: Live dashboard updates and bidirectional communication
- **Enterprise Architecture**: Microservices with GORM, PostgreSQL, and Redis
- **Advanced Security**: JWT authentication, rate limiting, and comprehensive security scanning
- **CLI & REST API**: Complete command-line interface and RESTful API
- **Monitoring & Analytics**: Built-in metrics collection and system health monitoring

## 🏗️ Architecture

### Core Components

- **HTTP Server** (Gin) - High-performance web server and REST API
- **WebSocket Hub** - Real-time bidirectional communication
- **R&D Module** - Self-learning system with agent coordination
- **Database Layer** (GORM + PostgreSQL) - Robust data persistence
- **CLI Interface** (Cobra) - Command-line management tools
- **Monitoring System** - Comprehensive metrics and health monitoring

## 🛠️ Installation

### Prerequisites

- **Go 1.22+** (recommended: Go 1.23)
- **PostgreSQL 13+**
- **Redis 6+** (optional, for caching)
- Redis 6+

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/your-username/KaskManager.git
cd KaskManager
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Build the applications:
```bash
make build
```

5. Start the platform:
```bash
make run
```

## 🔧 Configuration

### Environment Variables

Key environment variables (see `.env.example` for complete list):

- `PORT`: API server port (default: 8080)
- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string
- `JWT_SECRET`: JWT signing secret
- `OPENAI_API_KEY`: OpenAI API key for AI features
- `ANTHROPIC_API_KEY`: Anthropic API key for Claude integration

### R&D Module Configuration

The R&D module can be configured with various parameters:

- `RND_DORMANT_PERIOD`: Time before activation (default: 7 days)
- `RND_LEARNING_THRESHOLD`: Activation threshold (default: 0.7)
- `RND_MAX_SUGGESTIONS`: Maximum project suggestions (default: 3)

## 🚦 Usage

### CLI Commands

```bash
# Build and run CLI
make build-cli
./build/kaskman --help

# Build and run server
make build-server
./build/kaskmanager

# Development mode
make dev-cli    # Run CLI in development
make dev-server # Run server in development

# System management
make test       # Run tests
make lint       # Run linters
make format     # Format code
```

### API Endpoints

The REST API provides comprehensive endpoints:

- `GET /api/projects` - List projects
- `POST /api/projects` - Create project
- `GET /api/projects/:id/status` - Get project status
- `POST /api/projects/:id/start` - Start project
- `GET /api/system/status` - System status
- `GET /api/system/health` - Health check

### MCP Integration

The MCP server provides tools for Claude integration:

```bash
# Build and start MCP server
make build-server
./build/kaskmanager --mcp

# Available tools: project_create, project_list, system_status, etc.
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run benchmarks
make benchmark

# Run security scan
make security-scan
```

## 🔒 Security

### Authentication

The platform uses JWT-based authentication with:
- Secure password hashing (bcrypt)
- Token expiration and refresh
- Role-based access control

### Security Features

- Helmet.js for security headers
- Rate limiting to prevent abuse
- CORS configuration
- Input validation and sanitization
- SQL injection prevention

## 📊 Monitoring

### Health Checks

- `GET /health` - Basic health check
- `GET /api/system/health` - Comprehensive health status

### Metrics

- Prometheus metrics endpoint
- Grafana dashboards for visualization
- Real-time WebSocket monitoring

## 🔄 Development

### Scripts

- `make dev` - Start development server
- `make build` - Build production version
- `make lint` - Run Go linters
- `make format` - Format Go code
- `make tidy` - Tidy Go modules

### Development Workflow

1. Create feature branch
2. Make changes
3. Run tests and linting
4. Create pull request
5. Review and merge

## 📚 Documentation

- [API Documentation](./docs/api.md)
- [Architecture Guide](./docs/architecture.md)
- [Deployment Guide](./docs/deployment.md)
- [Development Guide](./docs/development.md)

## 🚀 Deployment

### Docker

```bash
# Build image
make docker-build

# Run container
make docker-run
```

### Kubernetes

```bash
# Deploy to Kubernetes
kubectl apply -f k8s/
```

### Production Considerations

- Use environment variables for configuration
- Set up SSL/TLS certificates
- Configure reverse proxy (nginx)
- Set up monitoring and logging
- Implement backup strategies

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🐛 Troubleshooting

### Common Issues

1. **Database connection issues**: Check PostgreSQL is running and credentials are correct
2. **Redis connection issues**: Ensure Redis is installed and running
3. **Authentication errors**: Verify JWT_SECRET is set and tokens are valid
4. **Port conflicts**: Check if the configured port is available

### Getting Help

- Create an issue on GitHub
- Check the documentation
- Review the logs in `./logs/`

## 📈 Roadmap

- [ ] Advanced AI integration
- [ ] Plugin system
- [ ] Multi-tenant support
- [ ] Mobile app
- [ ] Advanced analytics
- [ ] CI/CD pipeline integration

## 🏆 Acknowledgments

Built using modern technologies:
- Go 1.23 with Gin framework
- PostgreSQL & Redis with GORM
- WebSockets for real-time features
- JWT for authentication
- Cobra CLI framework