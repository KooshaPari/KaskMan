/**
 * Status Monitor
 * Provides real-time monitoring of system and project status
 */

import { EventEmitter } from 'events';
import os from 'os';
import { Logger } from './logger.js';
import { ProcessManager } from './process-manager.js';
import { MetricsCollector } from './metrics-collector.js';

class StatusMonitor extends EventEmitter {
  constructor(config = {}) {
    super();
    this.config = {
      updateInterval: config.updateInterval || 5000, // 5 seconds
      metricsRetention: config.metricsRetention || 7 * 24 * 60 * 60 * 1000, // 7 days
      alertThresholds: {
        cpu: config.alertThresholds?.cpu || 80,
        memory: config.alertThresholds?.memory || 85,
        disk: config.alertThresholds?.disk || 90,
        ...config.alertThresholds,
      },
      ...config,
    };

    this.logger = new Logger('StatusMonitor');
    this.processManager = new ProcessManager();
    this.metricsCollector = new MetricsCollector();

    this.monitoring = false;
    this.watchers = new Map();
    this.lastSystemStatus = null;
    this.projectStatuses = new Map();
    this.alerts = new Map();
  }

  async initialize() {
    try {
      await this.processManager.initialize();
      await this.metricsCollector.initialize();

      // Start monitoring
      this.startMonitoring();

      this.logger.info('StatusMonitor initialized successfully');
    } catch (error) {
      this.logger.error('Failed to initialize StatusMonitor:', error);
      throw error;
    }
  }

  startMonitoring() {
    if (this.monitoring) return;

    this.monitoring = true;
    this.monitoringInterval = setInterval(() => {
      this.collectSystemStatus();
      this.collectProjectStatuses();
      this.checkAlerts();
    }, this.config.updateInterval);

    this.logger.info('Started system monitoring');
  }

  stopMonitoring() {
    if (!this.monitoring) return;

    this.monitoring = false;
    if (this.monitoringInterval) {
      clearInterval(this.monitoringInterval);
    }

    this.logger.info('Stopped system monitoring');
  }

  async collectSystemStatus() {
    try {
      const systemStatus = {
        timestamp: new Date().toISOString(),
        cpu: await this.getCPUUsage(),
        memory: this.getMemoryUsage(),
        disk: await this.getDiskUsage(),
        loadAverage: os.loadavg(),
        uptime: os.uptime(),
        platform: os.platform(),
        arch: os.arch(),
        nodeVersion: process.version,
        activeProjects: this.processManager.getActiveProjectsCount(),
        networkInterfaces: this.getNetworkInterfaces(),
        processes: await this.getSystemProcesses(),
      };

      this.lastSystemStatus = systemStatus;
      this.emit('system:status', systemStatus);

      // Store metrics
      await this.metricsCollector.recordSystemMetrics(systemStatus);
    } catch (error) {
      this.logger.error('Failed to collect system status:', error);
    }
  }

  async collectProjectStatuses() {
    try {
      const runningProjects = await this.processManager.getRunningProjects();

      for (const [projectId, processInfo] of runningProjects) {
        const projectStatus = await this.getProjectStatusDetailed(
          projectId,
          processInfo
        );
        this.projectStatuses.set(projectId, projectStatus);
        this.emit('project:status', projectStatus);
      }
    } catch (error) {
      this.logger.error('Failed to collect project statuses:', error);
    }
  }

  async getProjectStatusDetailed(projectId, processInfo) {
    try {
      const status = {
        projectId,
        timestamp: new Date().toISOString(),
        process: {
          pid: processInfo.pid,
          status: processInfo.status,
          startTime: processInfo.startTime,
          uptime: Date.now() - processInfo.startTime,
          cpu: await this.getProcessCPUUsage(processInfo.pid),
          memory: await this.getProcessMemoryUsage(processInfo.pid),
          port: processInfo.port,
          url: processInfo.url,
        },
        health: await this.checkProjectHealth(projectId, processInfo),
        logs: await this.getRecentLogs(projectId, 10),
      };

      return status;
    } catch (error) {
      this.logger.error(
        `Failed to get detailed status for project ${projectId}:`,
        error
      );
      return {
        projectId,
        timestamp: new Date().toISOString(),
        error: error.message,
      };
    }
  }

  async checkProjectHealth(projectId, processInfo) {
    try {
      const health = {
        healthy: true,
        checks: {
          process: false,
          port: false,
          response: false,
        },
        issues: [],
      };

      // Check if process is running
      if (processInfo.status === 'running') {
        health.checks.process = true;
      } else {
        health.healthy = false;
        health.issues.push('Process not running');
      }

      // Check if port is accessible
      if (processInfo.port && (await this.isPortAccessible(processInfo.port))) {
        health.checks.port = true;
      } else if (processInfo.port) {
        health.healthy = false;
        health.issues.push(`Port ${processInfo.port} not accessible`);
      }

      // Check if application responds
      if (processInfo.url && (await this.isURLResponding(processInfo.url))) {
        health.checks.response = true;
      } else if (processInfo.url) {
        health.healthy = false;
        health.issues.push(`URL ${processInfo.url} not responding`);
      }

      return health;
    } catch (error) {
      return {
        healthy: false,
        checks: { process: false, port: false, response: false },
        issues: [`Health check failed: ${error.message}`],
      };
    }
  }

  async isPortAccessible(port) {
    try {
      const net = await import('net');
      return new Promise((resolve) => {
        const socket = new net.Socket();
        socket.setTimeout(1000);

        socket.on('connect', () => {
          socket.destroy();
          resolve(true);
        });

        socket.on('timeout', () => {
          socket.destroy();
          resolve(false);
        });

        socket.on('error', () => {
          resolve(false);
        });

        socket.connect(port, 'localhost');
      });
    } catch (error) {
      return false;
    }
  }

  async isURLResponding(url) {
    try {
      const response = await fetch(url, {
        method: 'HEAD',
        timeout: 5000,
      });
      return response.ok;
    } catch (error) {
      return false;
    }
  }

  async getCPUUsage() {
    return new Promise((resolve) => {
      const startUsage = process.cpuUsage();
      const startTime = Date.now();

      setTimeout(() => {
        const endUsage = process.cpuUsage(startUsage);
        const endTime = Date.now();

        const totalUsage = endUsage.user + endUsage.system;
        const totalTime = (endTime - startTime) * 1000; // Convert to microseconds

        const cpuPercent = (totalUsage / totalTime) * 100;
        resolve(Math.round(cpuPercent * 100) / 100);
      }, 100);
    });
  }

  getMemoryUsage() {
    const totalMemory = os.totalmem();
    const freeMemory = os.freemem();
    const usedMemory = totalMemory - freeMemory;

    return {
      total: totalMemory,
      free: freeMemory,
      used: usedMemory,
      percentage: Math.round((usedMemory / totalMemory) * 100 * 100) / 100,
    };
  }

  async getDiskUsage() {
    // This is a simplified version - in production, use a proper disk usage library
    return {
      total: 100 * 1024 * 1024 * 1024, // 100GB placeholder
      free: 50 * 1024 * 1024 * 1024, // 50GB placeholder
      used: 50 * 1024 * 1024 * 1024, // 50GB placeholder
      percentage: 50,
    };
  }

  getNetworkInterfaces() {
    const interfaces = os.networkInterfaces();
    const result = {};

    for (const [name, addresses] of Object.entries(interfaces)) {
      result[name] = addresses
        .filter((addr) => !addr.internal)
        .map((addr) => ({
          address: addr.address,
          family: addr.family,
          netmask: addr.netmask,
        }));
    }

    return result;
  }

  async getSystemProcesses() {
    try {
      // This is a simplified version - in production, use a proper process listing library
      return {
        total: 100, // placeholder
        nodeProcesses: 5, // placeholder
        platformProcesses: await this.processManager.getProcessCount(),
      };
    } catch (error) {
      return {
        total: 0,
        nodeProcesses: 0,
        platformProcesses: 0,
      };
    }
  }

  async getProcessCPUUsage(_pid) {
    // eslint-disable-line no-unused-vars
    try {
      // This is a simplified version - in production, use a proper process monitoring library
      // TODO: Use _pid parameter to get actual process CPU usage
      return Math.random() * 20; // placeholder
    } catch (error) {
      return 0;
    }
  }

  async getProcessMemoryUsage(_pid) {
    // eslint-disable-line no-unused-vars
    try {
      // This is a simplified version - in production, use a proper process monitoring library
      // TODO: Use _pid parameter to get actual process memory usage
      return {
        rss: Math.random() * 100 * 1024 * 1024, // placeholder
        heapTotal: Math.random() * 50 * 1024 * 1024, // placeholder
        heapUsed: Math.random() * 30 * 1024 * 1024, // placeholder
        external: Math.random() * 10 * 1024 * 1024, // placeholder
      };
    } catch (error) {
      return {
        rss: 0,
        heapTotal: 0,
        heapUsed: 0,
        external: 0,
      };
    }
  }

  async getRecentLogs(projectId, lines = 10) {
    try {
      return await this.processManager.getProjectLogs(projectId, { lines });
    } catch (error) {
      return [];
    }
  }

  checkAlerts() {
    if (!this.lastSystemStatus) return;

    const alerts = [];

    // CPU alert
    if (this.lastSystemStatus.cpu > this.config.alertThresholds.cpu) {
      alerts.push({
        type: 'cpu',
        level: 'warning',
        message: `High CPU usage: ${this.lastSystemStatus.cpu}%`,
        threshold: this.config.alertThresholds.cpu,
        current: this.lastSystemStatus.cpu,
        timestamp: new Date().toISOString(),
      });
    }

    // Memory alert
    if (
      this.lastSystemStatus.memory.percentage >
      this.config.alertThresholds.memory
    ) {
      alerts.push({
        type: 'memory',
        level: 'warning',
        message: `High memory usage: ${this.lastSystemStatus.memory.percentage}%`,
        threshold: this.config.alertThresholds.memory,
        current: this.lastSystemStatus.memory.percentage,
        timestamp: new Date().toISOString(),
      });
    }

    // Disk alert
    if (
      this.lastSystemStatus.disk.percentage > this.config.alertThresholds.disk
    ) {
      alerts.push({
        type: 'disk',
        level: 'warning',
        message: `High disk usage: ${this.lastSystemStatus.disk.percentage}%`,
        threshold: this.config.alertThresholds.disk,
        current: this.lastSystemStatus.disk.percentage,
        timestamp: new Date().toISOString(),
      });
    }

    // Emit alerts
    for (const alert of alerts) {
      this.emit('alert', alert);
      this.alerts.set(`${alert.type}-${alert.timestamp}`, alert);
    }

    // Clean up old alerts
    this.cleanupOldAlerts();
  }

  cleanupOldAlerts() {
    const now = Date.now();
    const maxAge = 24 * 60 * 60 * 1000; // 24 hours

    for (const [key, alert] of this.alerts.entries()) {
      if (now - new Date(alert.timestamp).getTime() > maxAge) {
        this.alerts.delete(key);
      }
    }
  }

  // Public API methods
  async getSystemStatus(detailed = false) {
    if (detailed) {
      await this.collectSystemStatus();
    }

    return (
      this.lastSystemStatus || {
        timestamp: new Date().toISOString(),
        status: 'unknown',
      }
    );
  }

  async getProjectStatus(projectId) {
    if (!projectId) {
      // Return all project statuses
      return Array.from(this.projectStatuses.values());
    }

    const status = this.projectStatuses.get(projectId);
    if (!status) {
      // Try to get fresh status
      const runningProjects = await this.processManager.getRunningProjects();
      const processInfo = runningProjects.get(projectId);

      if (processInfo) {
        return await this.getProjectStatusDetailed(projectId, processInfo);
      }

      throw new Error(`Project status not found: ${projectId}`);
    }

    return status;
  }

  async getHealthCheck(includeServices = true) {
    const health = {
      timestamp: new Date().toISOString(),
      overall: {
        healthy: true,
        status: 'healthy',
      },
      services: {},
    };

    if (includeServices) {
      // System health
      const systemStatus = await this.getSystemStatus(true);
      health.services.system = {
        healthy: systemStatus.cpu < 90 && systemStatus.memory.percentage < 90,
        status:
          systemStatus.cpu < 90 && systemStatus.memory.percentage < 90
            ? 'healthy'
            : 'degraded',
        details: `CPU: ${systemStatus.cpu}%, Memory: ${systemStatus.memory.percentage}%`,
      };

      // Process manager health
      health.services.processManager = {
        healthy: this.processManager.isHealthy(),
        status: this.processManager.isHealthy() ? 'healthy' : 'unhealthy',
        details: `Active projects: ${this.processManager.getActiveProjectsCount()}`,
      };

      // Check overall health
      health.overall.healthy = Object.values(health.services).every(
        (service) => service.healthy
      );
      health.overall.status = health.overall.healthy ? 'healthy' : 'degraded';
    }

    return health;
  }

  async getSystemMetrics(options = {}) {
    const timeRange = options.timeRange || '1h';
    const metrics = options.metrics || ['cpu', 'memory', 'disk', 'network'];

    return await this.metricsCollector.getMetrics(timeRange, metrics);
  }

  async triggerMaintenance(options = {}) {
    const tasks = options.tasks || ['cleanup', 'optimize', 'check'];
    const schedule = options.schedule || 'now';

    const result = {
      triggered: new Date().toISOString(),
      tasks,
      schedule,
      status: 'scheduled',
    };

    // Emit maintenance event
    this.emit('maintenance:triggered', result);

    // Execute maintenance tasks
    if (schedule === 'now') {
      await this.executeMaintenance(tasks);
      result.status = 'completed';
    }

    return result;
  }

  async executeMaintenance(tasks) {
    for (const task of tasks) {
      try {
        switch (task) {
          case 'cleanup':
            await this.cleanupLogs();
            await this.cleanupMetrics();
            break;
          case 'optimize':
            await this.optimizeSystem();
            break;
          case 'check':
            await this.systemHealthCheck();
            break;
          default:
            this.logger.warn(`Unknown maintenance task: ${task}`);
        }
      } catch (error) {
        this.logger.error(`Maintenance task failed: ${task}`, error);
      }
    }
  }

  async cleanupLogs() {
    // Cleanup old logs
    this.logger.info('Cleaning up old logs...');
  }

  async cleanupMetrics() {
    // Cleanup old metrics
    await this.metricsCollector.cleanup();
  }

  async optimizeSystem() {
    // System optimization
    this.logger.info('Optimizing system...');
  }

  async systemHealthCheck() {
    // Comprehensive health check
    const health = await this.getHealthCheck(true);
    this.logger.info('System health check completed', health);
  }

  // Watcher methods
  watchProject(projectId) {
    const watcher = {
      projectId,
      active: true,
      emit: (event, data) => this.emit(event, data),
      stop: () => {
        watcher.active = false;
        this.watchers.delete(projectId);
      },
    };

    this.watchers.set(projectId, watcher);

    // Set up project-specific monitoring
    const projectInterval = setInterval(() => {
      if (!watcher.active) {
        clearInterval(projectInterval);
        return;
      }

      this.getProjectStatus(projectId)
        .then((status) => watcher.emit('status', status))
        .catch((error) => watcher.emit('error', error));
    }, 2000);

    return watcher;
  }

  watchSystem() {
    const watcher = {
      active: true,
      emit: (event, data) => this.emit(event, data),
      stop: () => {
        watcher.active = false;
        this.watchers.delete('system');
      },
    };

    this.watchers.set('system', watcher);

    // Set up system-specific monitoring
    const systemInterval = setInterval(() => {
      if (!watcher.active) {
        clearInterval(systemInterval);
        return;
      }

      this.getSystemStatus(true)
        .then((status) => watcher.emit('status', status))
        .catch((error) => watcher.emit('error', error));
    }, 1000);

    return watcher;
  }

  async stop() {
    try {
      this.stopMonitoring();

      // Stop all watchers
      for (const watcher of this.watchers.values()) {
        watcher.stop();
      }

      await this.processManager.stop();
      await this.metricsCollector.stop();

      this.logger.info('StatusMonitor stopped successfully');
    } catch (error) {
      this.logger.error('Error stopping StatusMonitor:', error);
      throw error;
    }
  }
}

export { StatusMonitor };
