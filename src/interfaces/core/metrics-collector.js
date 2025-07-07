/**
 * Metrics Collector
 * Collects and stores system performance metrics
 */

import { EventEmitter } from 'events';
import { promises as fs } from 'fs';
import path from 'path';
import { Logger } from './logger.js';

class MetricsCollector extends EventEmitter {
  constructor(config = {}) {
    super();
    this.config = {
      metricsFile: config.metricsFile || './data/metrics.json',
      retentionPeriod: config.retentionPeriod || 7 * 24 * 60 * 60 * 1000, // 7 days
      ...config,
    };

    this.logger = new Logger('MetricsCollector');
    this.metrics = [];
  }

  async initialize() {
    try {
      await this.loadMetrics();
      this.logger.info('MetricsCollector initialized successfully');
    } catch (error) {
      this.logger.error('Failed to initialize MetricsCollector:', error);
      throw error;
    }
  }

  async loadMetrics() {
    try {
      const metricsData = await fs.readFile(this.config.metricsFile, 'utf8');
      this.metrics = JSON.parse(metricsData);
      this.logger.info(`Loaded ${this.metrics.length} metrics`);
    } catch (error) {
      if (error.code === 'ENOENT') {
        this.logger.info('No metrics file found, starting with empty metrics');
        await this.saveMetrics();
      } else {
        this.logger.error('Failed to load metrics:', error);
        throw error;
      }
    }
  }

  async saveMetrics() {
    try {
      const metricsDir = path.dirname(this.config.metricsFile);
      await fs.mkdir(metricsDir, { recursive: true });

      await fs.writeFile(
        this.config.metricsFile,
        JSON.stringify(this.metrics, null, 2)
      );
    } catch (error) {
      this.logger.error('Failed to save metrics:', error);
      throw error;
    }
  }

  async recordSystemMetrics(systemStatus) {
    try {
      const metric = {
        timestamp: systemStatus.timestamp,
        type: 'system',
        data: {
          cpu: systemStatus.cpu,
          memory: systemStatus.memory,
          disk: systemStatus.disk,
          loadAverage: systemStatus.loadAverage,
          uptime: systemStatus.uptime,
          activeProjects: systemStatus.activeProjects,
        },
      };

      this.metrics.push(metric);
      await this.saveMetrics();

      this.emit('metric:recorded', metric);
    } catch (error) {
      this.logger.error('Failed to record system metrics:', error);
    }
  }

  async getMetrics(timeRange = '1h', metricTypes = []) {
    try {
      const now = Date.now();
      const ranges = {
        '1h': 60 * 60 * 1000,
        '24h': 24 * 60 * 60 * 1000,
        '7d': 7 * 24 * 60 * 60 * 1000,
        '30d': 30 * 24 * 60 * 60 * 1000,
      };

      const rangeMs = ranges[timeRange] || ranges['1h'];
      const startTime = now - rangeMs;

      let filteredMetrics = this.metrics.filter((metric) => {
        const metricTime = new Date(metric.timestamp).getTime();
        return metricTime >= startTime;
      });

      if (metricTypes.length > 0) {
        filteredMetrics = filteredMetrics.filter((metric) =>
          metricTypes.includes(metric.type)
        );
      }

      return filteredMetrics;
    } catch (error) {
      this.logger.error('Failed to get metrics:', error);
      throw error;
    }
  }

  async cleanup() {
    try {
      const now = Date.now();
      const cutoffTime = now - this.config.retentionPeriod;

      const initialCount = this.metrics.length;
      this.metrics = this.metrics.filter((metric) => {
        const metricTime = new Date(metric.timestamp).getTime();
        return metricTime >= cutoffTime;
      });

      const removedCount = initialCount - this.metrics.length;
      if (removedCount > 0) {
        await this.saveMetrics();
        this.logger.info(`Cleaned up ${removedCount} old metrics`);
      }
    } catch (error) {
      this.logger.error('Failed to cleanup metrics:', error);
    }
  }

  async stop() {
    try {
      await this.saveMetrics();
      this.logger.info('MetricsCollector stopped successfully');
    } catch (error) {
      this.logger.error('Error stopping MetricsCollector:', error);
      throw error;
    }
  }
}

export { MetricsCollector };
