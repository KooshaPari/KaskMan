/**
 * REST API Server for R&D Platform
 * Provides HTTP endpoints for remote access to platform functionality
 */

import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import rateLimit from 'express-rate-limit';
import compression from 'compression';
import { createServer } from 'http';
import { Server as SocketServer } from 'socket.io';
import { ProjectManager } from '../core/project-manager.js';
import { StatusMonitor } from '../core/status-monitor.js';
import { AuthManager } from '../core/auth-manager.js';
import { Logger } from '../core/logger.js';
import { errorHandler, notFoundHandler } from './middleware/error-handler.js';
import { authMiddleware } from './middleware/auth-middleware.js';
import { validateRequest } from './middleware/validation.js';
import { projectRoutes } from './routes/projects.js';
import { systemRoutes } from './routes/system.js';
import { authRoutes } from './routes/auth.js';
import { webhookRoutes } from './routes/webhooks.js';

class APIServer {
  constructor(config = {}) {
    this.config = {
      port: config.port || process.env.PORT || 8080,
      host: config.host || process.env.HOST || '0.0.0.0',
      cors: config.cors || { origin: true },
      rateLimit: config.rateLimit || { windowMs: 15 * 60 * 1000, max: 100 },
      ...config,
    };

    this.app = express();
    this.httpServer = createServer(this.app);
    this.io = new SocketServer(this.httpServer, {
      cors: this.config.cors,
    });

    this.projectManager = new ProjectManager();
    this.statusMonitor = new StatusMonitor();
    this.authManager = new AuthManager();
    this.logger = new Logger('APIServer');

    this.setupMiddleware();
    this.setupRoutes();
    this.setupWebSocket();
    this.setupErrorHandling();
  }

  setupMiddleware() {
    // Security middleware
    this.app.use(
      helmet({
        contentSecurityPolicy: {
          directives: {
            defaultSrc: ["'self'"],
            styleSrc: ["'self'", "'unsafe-inline'"],
            scriptSrc: ["'self'"],
            imgSrc: ["'self'", 'data:', 'https:'],
          },
        },
      })
    );

    // CORS
    this.app.use(cors(this.config.cors));

    // Rate limiting
    const limiter = rateLimit(this.config.rateLimit);
    this.app.use(limiter);

    // Compression
    this.app.use(compression());

    // Body parsing
    this.app.use(express.json({ limit: '10mb' }));
    this.app.use(express.urlencoded({ extended: true, limit: '10mb' }));

    // Logging
    this.app.use((req, res, next) => {
      this.logger.info(`${req.method} ${req.path}`, {
        ip: req.ip,
        userAgent: req.get('User-Agent'),
        requestId: req.id,
      });
      next();
    });

    // Request validation
    this.app.use(validateRequest);
  }

  setupRoutes() {
    // Health check endpoint
    this.app.get('/health', (req, res) => {
      res.json({
        status: 'healthy',
        timestamp: new Date().toISOString(),
        version: process.env.npm_package_version || '1.0.0',
        uptime: process.uptime(),
      });
    });

    // API documentation
    this.app.get('/api/docs', (req, res) => {
      res.json({
        title: 'R&D Platform API',
        version: '1.0.0',
        description:
          'REST API for R&D Platform project management and system control',
        endpoints: {
          auth: {
            'POST /api/auth/login': 'Authenticate user',
            'POST /api/auth/logout': 'Logout user',
            'GET /api/auth/me': 'Get current user info',
            'POST /api/auth/refresh': 'Refresh access token',
          },
          projects: {
            'GET /api/projects': 'List projects',
            'POST /api/projects': 'Create project',
            'GET /api/projects/:id': 'Get project details',
            'PUT /api/projects/:id': 'Update project',
            'DELETE /api/projects/:id': 'Delete project',
            'POST /api/projects/:id/start': 'Start project',
            'POST /api/projects/:id/stop': 'Stop project',
            'GET /api/projects/:id/status': 'Get project status',
            'GET /api/projects/:id/logs': 'Get project logs',
          },
          system: {
            'GET /api/system/status': 'Get system status',
            'GET /api/system/health': 'Health check',
            'GET /api/system/metrics': 'Get system metrics',
            'POST /api/system/maintenance': 'Trigger maintenance',
          },
          webhooks: {
            'POST /api/webhooks/github': 'GitHub webhook',
            'POST /api/webhooks/deploy': 'Deployment webhook',
            'GET /api/webhooks': 'List webhooks',
          },
        },
      });
    });

    // API routes
    this.app.use('/api/auth', authRoutes);
    this.app.use('/api/projects', authMiddleware, projectRoutes);
    this.app.use('/api/system', authMiddleware, systemRoutes);
    this.app.use('/api/webhooks', webhookRoutes);

    // WebSocket status endpoint
    this.app.get('/api/socket/status', authMiddleware, (req, res) => {
      res.json({
        connected: this.io.engine.clientsCount,
        rooms: Object.keys(this.io.sockets.adapter.rooms),
      });
    });
  }

  setupWebSocket() {
    this.io.use(async (socket, next) => {
      try {
        const token = socket.handshake.auth.token;
        const user = await this.authManager.verifyToken(token);
        socket.user = user;
        next();
      } catch (error) {
        next(new Error('Authentication failed'));
      }
    });

    this.io.on('connection', (socket) => {
      this.logger.info(`WebSocket connected: ${socket.id}`, {
        userId: socket.user.id,
        username: socket.user.username,
      });

      // Join user-specific room
      socket.join(`user:${socket.user.id}`);

      // Project status subscriptions
      socket.on('subscribe:project', (projectId) => {
        socket.join(`project:${projectId}`);
        this.logger.info(`User subscribed to project ${projectId}`, {
          userId: socket.user.id,
          socketId: socket.id,
        });
      });

      socket.on('unsubscribe:project', (projectId) => {
        socket.leave(`project:${projectId}`);
        this.logger.info(`User unsubscribed from project ${projectId}`, {
          userId: socket.user.id,
          socketId: socket.id,
        });
      });

      // System status subscription
      socket.on('subscribe:system', () => {
        socket.join('system');
        this.logger.info(`User subscribed to system status`, {
          userId: socket.user.id,
          socketId: socket.id,
        });
      });

      socket.on('unsubscribe:system', () => {
        socket.leave('system');
        this.logger.info(`User unsubscribed from system status`, {
          userId: socket.user.id,
          socketId: socket.id,
        });
      });

      // Real-time project operations
      socket.on('project:start', async (projectId) => {
        try {
          await this.projectManager.startProject(projectId);
          this.io
            .to(`project:${projectId}`)
            .emit('project:started', { projectId });
        } catch (error) {
          socket.emit('error', { message: error.message });
        }
      });

      socket.on('project:stop', async (projectId) => {
        try {
          await this.projectManager.stopProject(projectId);
          this.io
            .to(`project:${projectId}`)
            .emit('project:stopped', { projectId });
        } catch (error) {
          socket.emit('error', { message: error.message });
        }
      });

      socket.on('disconnect', () => {
        this.logger.info(`WebSocket disconnected: ${socket.id}`, {
          userId: socket.user.id,
        });
      });
    });

    // Status monitoring integration
    this.statusMonitor.on('project:status', (data) => {
      this.io.to(`project:${data.projectId}`).emit('project:status', data);
    });

    this.statusMonitor.on('system:status', (data) => {
      this.io.to('system').emit('system:status', data);
    });

    this.statusMonitor.on('project:log', (data) => {
      this.io.to(`project:${data.projectId}`).emit('project:log', data);
    });
  }

  setupErrorHandling() {
    // 404 handler
    this.app.use(notFoundHandler);

    // Error handler
    this.app.use(errorHandler);

    // Unhandled promise rejections
    process.on('unhandledRejection', (reason, promise) => {
      this.logger.error('Unhandled Rejection at:', promise, 'reason:', reason);
      this.gracefulShutdown();
    });

    // Uncaught exceptions
    process.on('uncaughtException', (error) => {
      this.logger.error('Uncaught Exception:', error);
      this.gracefulShutdown();
    });

    // Graceful shutdown
    process.on('SIGTERM', () => {
      this.logger.info('SIGTERM received, shutting down gracefully');
      this.gracefulShutdown();
    });

    process.on('SIGINT', () => {
      this.logger.info('SIGINT received, shutting down gracefully');
      this.gracefulShutdown();
    });
  }

  async start() {
    try {
      await this.projectManager.initialize();
      await this.statusMonitor.initialize();

      this.httpServer.listen(this.config.port, this.config.host, () => {
        this.logger.info(`API Server started`, {
          port: this.config.port,
          host: this.config.host,
          environment: process.env.NODE_ENV || 'development',
        });
      });

      return {
        url: `http://${this.config.host}:${this.config.port}`,
        port: this.config.port,
        host: this.config.host,
      };
    } catch (error) {
      this.logger.error('Failed to start API server:', error);
      throw error;
    }
  }

  async stop() {
    try {
      this.logger.info('Stopping API server...');

      // Stop accepting new connections
      this.httpServer.close(() => {
        this.logger.info('HTTP server closed');
      });

      // Close WebSocket connections
      this.io.close(() => {
        this.logger.info('WebSocket server closed');
      });

      // Stop monitoring
      await this.statusMonitor.stop();

      this.logger.info('API server stopped successfully');
    } catch (error) {
      this.logger.error('Error stopping API server:', error);
      throw error;
    }
  }

  gracefulShutdown() {
    this.logger.info('Starting graceful shutdown...');

    setTimeout(() => {
      this.logger.error('Forceful shutdown after timeout');
      process.exit(1);
    }, 30000); // 30 seconds timeout

    this.stop()
      .then(() => {
        this.logger.info('Graceful shutdown completed');
        process.exit(0);
      })
      .catch((error) => {
        this.logger.error('Error during graceful shutdown:', error);
        process.exit(1);
      });
  }

  // Public API for external access
  getExpressApp() {
    return this.app;
  }

  getHttpServer() {
    return this.httpServer;
  }

  getSocketIO() {
    return this.io;
  }
}

export { APIServer };

// CLI integration
if (import.meta.url === `file://${process.argv[1]}`) {
  const server = new APIServer({
    port: process.env.PORT || 8080,
    host: process.env.HOST || '0.0.0.0',
  });

  server.start().catch((error) => {
    console.error('Failed to start server:', error);
    process.exit(1);
  });
}
