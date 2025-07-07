/**
 * API Client
 * Handles API server management and client operations
 */

import { spawn } from 'child_process';
import { promises as fs } from 'fs';
import path from 'path';
import { Logger } from './logger.js';
import { ProcessManager } from './process-manager.js';

class APIClient {
  constructor(config = {}) {
    this.config = {
      serverPath: config.serverPath || './src/interfaces/api/server.js',
      defaultPort: config.defaultPort || 8080,
      defaultHost: config.defaultHost || 'localhost',
      timeout: config.timeout || 30000,
      retries: config.retries || 3,
      ...config,
    };

    this.logger = new Logger('APIClient');
    this.processManager = new ProcessManager();
    this.serverProcess = null;
    this.serverConfig = null;
  }

  async startServer(serverConfig = {}) {
    try {
      if (this.serverProcess) {
        throw new Error('Server is already running');
      }

      const config = {
        port: serverConfig.port || this.config.defaultPort,
        host: serverConfig.host || this.config.defaultHost,
        daemon: serverConfig.daemon || false,
        ...serverConfig,
      };

      this.serverConfig = config;

      // Check if server file exists
      const serverPath = path.resolve(this.config.serverPath);
      await fs.access(serverPath);

      // Start server process
      const env = {
        ...process.env,
        PORT: config.port.toString(),
        HOST: config.host,
        NODE_ENV: process.env.NODE_ENV || 'development',
      };

      this.serverProcess = spawn('node', [serverPath], {
        env,
        stdio: config.daemon ? 'ignore' : 'inherit',
        detached: config.daemon,
      });

      // Handle process events
      this.serverProcess.on('error', (error) => {
        this.logger.error('Server process error:', error);
      });

      this.serverProcess.on('exit', (code, signal) => {
        this.logger.info(
          `Server process exited with code ${code} and signal ${signal}`
        );
        this.serverProcess = null;
        this.serverConfig = null;
      });

      // Wait for server to start
      await this.waitForServer(config.host, config.port);

      const serverInfo = {
        pid: this.serverProcess.pid,
        port: config.port,
        host: config.host,
        url: `http://${config.host}:${config.port}`,
        daemon: config.daemon,
      };

      this.logger.info(`API server started successfully`, serverInfo);

      return serverInfo;
    } catch (error) {
      this.logger.error('Failed to start API server:', error);
      throw error;
    }
  }

  async stopServer() {
    try {
      if (!this.serverProcess) {
        throw new Error('Server is not running');
      }

      this.logger.info('Stopping API server...');

      // Graceful shutdown
      this.serverProcess.kill('SIGTERM');

      // Wait for graceful shutdown
      await new Promise((resolve) => {
        const timeout = setTimeout(() => {
          this.logger.warn('Forcing server shutdown...');
          this.serverProcess.kill('SIGKILL');
          resolve();
        }, 10000);

        this.serverProcess.on('exit', () => {
          clearTimeout(timeout);
          resolve();
        });
      });

      this.serverProcess = null;
      this.serverConfig = null;

      this.logger.info('API server stopped successfully');
    } catch (error) {
      this.logger.error('Failed to stop API server:', error);
      throw error;
    }
  }

  async waitForServer(host, port, timeout = 30000) {
    const startTime = Date.now();

    while (Date.now() - startTime < timeout) {
      try {
        await this.checkServerHealth(host, port);
        return;
      } catch (error) {
        // Server not ready yet, wait and retry
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }
    }

    throw new Error(`Server failed to start within ${timeout}ms`);
  }

  async checkServerHealth(host, port) {
    try {
      const response = await fetch(`http://${host}:${port}/health`, {
        method: 'GET',
        timeout: 5000,
      });

      if (!response.ok) {
        throw new Error(`Health check failed with status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      throw new Error(`Health check failed: ${error.message}`);
    }
  }

  async getServerStatus() {
    try {
      if (!this.serverProcess) {
        return {
          running: false,
          status: 'stopped',
        };
      }

      const health = await this.checkServerHealth(
        this.serverConfig.host,
        this.serverConfig.port
      );

      return {
        running: true,
        status: 'running',
        pid: this.serverProcess.pid,
        config: this.serverConfig,
        health,
      };
    } catch (error) {
      return {
        running: false,
        status: 'unhealthy',
        error: error.message,
      };
    }
  }

  async restartServer() {
    try {
      if (this.serverProcess) {
        await this.stopServer();
      }

      return await this.startServer(this.serverConfig);
    } catch (error) {
      this.logger.error('Failed to restart server:', error);
      throw error;
    }
  }

  isServerRunning() {
    return !!this.serverProcess;
  }

  getServerConfig() {
    return this.serverConfig;
  }

  // Client methods for making requests to the API
  async request(endpoint, options = {}) {
    if (!this.serverConfig) {
      throw new Error('Server is not running');
    }

    const url = `http://${this.serverConfig.host}:${this.serverConfig.port}${endpoint}`;
    const requestOptions = {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      timeout: options.timeout || this.config.timeout,
      ...options,
    };

    if (options.body && typeof options.body === 'object') {
      requestOptions.body = JSON.stringify(options.body);
    }

    let retries = options.retries || this.config.retries;
    let lastError;

    while (retries >= 0) {
      try {
        const response = await fetch(url, requestOptions);

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
          return await response.json();
        } else {
          return await response.text();
        }
      } catch (error) {
        lastError = error;
        retries--;

        if (retries >= 0) {
          this.logger.warn(
            `Request failed, retrying... (${retries} attempts left)`
          );
          await new Promise((resolve) => setTimeout(resolve, 1000));
        }
      }
    }

    throw lastError;
  }

  async get(endpoint, options = {}) {
    return this.request(endpoint, { ...options, method: 'GET' });
  }

  async post(endpoint, data, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'POST',
      body: data,
    });
  }

  async put(endpoint, data, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'PUT',
      body: data,
    });
  }

  async delete(endpoint, options = {}) {
    return this.request(endpoint, { ...options, method: 'DELETE' });
  }

  async stop() {
    try {
      if (this.serverProcess) {
        await this.stopServer();
      }

      this.logger.info('APIClient stopped successfully');
    } catch (error) {
      this.logger.error('Error stopping APIClient:', error);
      throw error;
    }
  }
}

export { APIClient };
