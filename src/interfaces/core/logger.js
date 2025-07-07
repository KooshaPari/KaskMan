/**
 * Logger
 * Provides structured logging with levels and formatting
 */

import { createWriteStream } from 'fs';
import { promises as fs } from 'fs';
import path from 'path';
import util from 'util';

class Logger {
  constructor(name, config = {}) {
    this.name = name;
    this.config = {
      level: config.level || process.env.LOG_LEVEL || 'info',
      format: config.format || 'json',
      console: config.console !== false,
      file: config.file || null,
      maxSize: config.maxSize || 10 * 1024 * 1024, // 10MB
      maxFiles: config.maxFiles || 5,
      ...config,
    };

    this.levels = {
      error: 0,
      warn: 1,
      info: 2,
      debug: 3,
      trace: 4,
    };

    this.colors = {
      error: '\x1b[31m', // Red
      warn: '\x1b[33m', // Yellow
      info: '\x1b[36m', // Cyan
      debug: '\x1b[32m', // Green
      trace: '\x1b[35m', // Magenta
    };

    this.reset = '\x1b[0m';
    this.fileStream = null;
    this.currentLogFile = null;
    this.currentLogSize = 0;

    this.setupFileLogging();
  }

  async setupFileLogging() {
    if (this.config.file) {
      try {
        const logDir = path.dirname(this.config.file);
        await fs.mkdir(logDir, { recursive: true });

        this.currentLogFile = this.config.file;
        this.fileStream = createWriteStream(this.currentLogFile, {
          flags: 'a',
        });

        // Get current file size
        try {
          const stats = await fs.stat(this.currentLogFile);
          this.currentLogSize = stats.size;
        } catch (error) {
          this.currentLogSize = 0;
        }
      } catch (error) {
        console.error('Failed to setup file logging:', error);
      }
    }
  }

  shouldLog(level) {
    const targetLevel = this.levels[this.config.level];
    const messageLevel = this.levels[level];
    return messageLevel <= targetLevel;
  }

  formatMessage(level, message, meta = {}) {
    const timestamp = new Date().toISOString();
    const logEntry = {
      timestamp,
      level,
      name: this.name,
      message,
      ...meta,
    };

    if (this.config.format === 'json') {
      return JSON.stringify(logEntry);
    } else {
      const metaStr =
        Object.keys(meta).length > 0 ? ` ${JSON.stringify(meta)}` : '';
      return `${timestamp} [${level.toUpperCase()}] ${this.name}: ${message}${metaStr}`;
    }
  }

  formatConsoleMessage(level, message, meta = {}) {
    const timestamp = new Date().toISOString();
    const color = this.colors[level] || '';
    const metaStr =
      Object.keys(meta).length > 0
        ? ` ${util.inspect(meta, { colors: true })}`
        : '';

    return `${color}${timestamp} [${level.toUpperCase()}] ${this.name}:${this.reset} ${message}${metaStr}`;
  }

  async rotateLogFile() {
    if (!this.fileStream || this.currentLogSize < this.config.maxSize) {
      return;
    }

    try {
      // Close current stream
      this.fileStream.end();

      // Rotate existing files
      for (let i = this.config.maxFiles - 1; i >= 1; i--) {
        const oldFile = `${this.currentLogFile}.${i}`;
        const newFile = `${this.currentLogFile}.${i + 1}`;

        try {
          await fs.access(oldFile);
          await fs.rename(oldFile, newFile);
        } catch (error) {
          // File doesn't exist, skip
        }
      }

      // Move current file to .1
      await fs.rename(this.currentLogFile, `${this.currentLogFile}.1`);

      // Create new file stream
      this.fileStream = createWriteStream(this.currentLogFile, { flags: 'a' });
      this.currentLogSize = 0;
    } catch (error) {
      console.error('Failed to rotate log file:', error);
    }
  }

  async writeToFile(formattedMessage) {
    if (!this.fileStream) return;

    try {
      await this.rotateLogFile();

      const messageWithNewline = formattedMessage + '\n';
      this.fileStream.write(messageWithNewline);
      this.currentLogSize += Buffer.byteLength(messageWithNewline);
    } catch (error) {
      console.error('Failed to write to log file:', error);
    }
  }

  log(level, message, meta = {}) {
    if (!this.shouldLog(level)) return;

    const formattedMessage = this.formatMessage(level, message, meta);

    // Console output
    if (this.config.console) {
      const consoleMessage = this.formatConsoleMessage(level, message, meta);
      console.log(consoleMessage);
    }

    // File output
    if (this.config.file) {
      this.writeToFile(formattedMessage);
    }
  }

  error(message, meta = {}) {
    this.log('error', message, meta);
  }

  warn(message, meta = {}) {
    this.log('warn', message, meta);
  }

  info(message, meta = {}) {
    this.log('info', message, meta);
  }

  debug(message, meta = {}) {
    this.log('debug', message, meta);
  }

  trace(message, meta = {}) {
    this.log('trace', message, meta);
  }

  child(name, additionalMeta = {}) {
    const childLogger = new Logger(`${this.name}:${name}`, this.config);
    childLogger.defaultMeta = { ...this.defaultMeta, ...additionalMeta };
    return childLogger;
  }

  close() {
    if (this.fileStream) {
      this.fileStream.end();
      this.fileStream = null;
    }
  }
}

export { Logger };
