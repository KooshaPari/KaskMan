# KaskManager R&D Platform

A persistent, always-on, self-improving utility and R&D platform designed for continuous research, development, and project management.

## 🚀 Features

- **Always-On Operation**: Zero-downtime, continuous operation with hot-swappable components
- **Self-Learning R&D Module**: Dormant-to-active progression with automatic project generation
- **Multiple Interfaces**: CLI, REST API, MCP server, and web dashboard
- **Real-Time Monitoring**: WebSocket-based status updates and monitoring
- **Microservices Architecture**: Scalable, fault-tolerant design
- **Advanced Security**: JWT authentication, rate limiting, CORS protection

## 📋 Architecture

The platform consists of several key components:

- **CLI Interface**: Command-line tools for project management and system control
- **API Server**: RESTful API with WebSocket support for real-time features
- **MCP Server**: Model Context Protocol server for Claude integration
- **R&D Module**: Self-learning system with pattern recognition and project generation
- **Dashboard**: Web-based and TUI interfaces for monitoring and management

## 🛠️ Installation

### Prerequisites

- Node.js 18+
- npm 9+
- PostgreSQL 13+
- Redis 6+

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/your-username/KaskManager.git
cd KaskManager
```

2. Install dependencies:
```bash
npm install
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Initialize the database:
```bash
npm run db:init
```

5. Start the platform:
```bash
npm start
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
# Authentication
rd-platform auth login
rd-platform auth status

# Project Management
rd-platform project create "My Project"
rd-platform project list
rd-platform project start <project-id>
rd-platform project status <project-id>

# System Management
rd-platform system status
rd-platform system health

# Server Management
rd-platform server start --port 8080
rd-platform server stop
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
# Start MCP server
npm run start:mcp

# Available tools: project_create, project_list, system_status, etc.
```

## 🧪 Testing

```bash
# Run all tests
npm test

# Run tests with coverage
npm run test:coverage

# Run tests in watch mode
npm run test:watch
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

- `npm run dev` - Start development server
- `npm run build` - Build production version
- `npm run lint` - Run ESLint
- `npm run format` - Format code with Prettier
- `npm run typecheck` - TypeScript type checking

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
docker build -t rd-platform .

# Run container
docker run -p 8080:8080 rd-platform
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
- Node.js & Express
- PostgreSQL & Redis
- Socket.io for real-time features
- JWT for authentication
- TypeScript for type safety