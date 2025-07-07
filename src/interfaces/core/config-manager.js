/**
 * Configuration Manager
 * Handles application configuration loading and management
 */

import { promises as fs } from 'fs';
import path from 'path';
import { Logger } from './logger.js';

class ConfigManager {
  constructor(config = {}) {
    this.config = {
      configFile: config.configFile || './config/app.json',
      environment: config.environment || process.env.NODE_ENV || 'development',
      ...config,
    };

    this.logger = new Logger('ConfigManager');
    this.configuration = {};
    this.watchers = new Map();
  }

  async initialize() {
    try {
      await this.loadConfiguration();
      this.logger.info('ConfigManager initialized successfully');
    } catch (error) {
      this.logger.error('Failed to initialize ConfigManager:', error);
      throw error;
    }
  }

  async loadConfiguration() {
    try {
      const configData = await fs.readFile(this.config.configFile, 'utf8');
      this.configuration = JSON.parse(configData);
      this.logger.info(`Loaded configuration from ${this.config.configFile}`);
    } catch (error) {
      if (error.code === 'ENOENT') {
        this.logger.info('No configuration file found, using defaults');
        await this.createDefaultConfiguration();
      } else {
        this.logger.error('Failed to load configuration:', error);
        throw error;
      }
    }
  }

  async createDefaultConfiguration() {
    const defaultConfig = {
      environment: this.config.environment,
      server: {
        port: 8080,
        host: '0.0.0.0',
      },
      database: {
        path: './data/database.json',
      },
      logging: {
        level: 'info',
        file: './logs/app.log',
      },
      security: {
        jwtSecret: 'your-secret-key-here',
        sessionTimeout: 86400000,
      },
    };

    this.configuration = defaultConfig;
    await this.saveConfiguration();
  }

  async saveConfiguration() {
    try {
      const configDir = path.dirname(this.config.configFile);
      await fs.mkdir(configDir, { recursive: true });

      await fs.writeFile(
        this.config.configFile,
        JSON.stringify(this.configuration, null, 2)
      );

      this.logger.info(`Configuration saved to ${this.config.configFile}`);
    } catch (error) {
      this.logger.error('Failed to save configuration:', error);
      throw error;
    }
  }

  get(key, defaultValue = null) {
    const keys = key.split('.');
    let value = this.configuration;

    for (const k of keys) {
      if (value && typeof value === 'object' && k in value) {
        value = value[k];
      } else {
        return defaultValue;
      }
    }

    return value;
  }

  set(key, value) {
    const keys = key.split('.');
    let target = this.configuration;

    for (let i = 0; i < keys.length - 1; i++) {
      const k = keys[i];
      if (!(k in target) || typeof target[k] !== 'object') {
        target[k] = {};
      }
      target = target[k];
    }

    target[keys[keys.length - 1]] = value;
  }

  has(key) {
    return this.get(key) !== null;
  }

  getAll() {
    return { ...this.configuration };
  }

  async reload() {
    await this.loadConfiguration();
  }

  async save() {
    await this.saveConfiguration();
  }
}

export { ConfigManager };
