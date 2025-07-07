/**
 * File Manager
 * Provides file system operations and utilities
 */

import { promises as fs } from 'fs';
import { Logger } from './logger.js';

class FileManager {
  constructor(config = {}) {
    this.config = {
      ...config,
    };

    this.logger = new Logger('FileManager');
  }

  async exists(filePath) {
    try {
      await fs.access(filePath);
      return true;
    } catch (error) {
      return false;
    }
  }

  async ensureDirectory(dirPath) {
    await fs.mkdir(dirPath, { recursive: true });
  }

  async removeDirectory(dirPath) {
    await fs.rmdir(dirPath, { recursive: true });
  }

  async copyFile(source, destination) {
    await fs.copyFile(source, destination);
  }

  async moveFile(source, destination) {
    await fs.rename(source, destination);
  }

  async readFile(filePath) {
    return await fs.readFile(filePath, 'utf8');
  }

  async writeFile(filePath, content) {
    await fs.writeFile(filePath, content);
  }

  async getFileStats(filePath) {
    return await fs.stat(filePath);
  }

  async listFiles(dirPath) {
    return await fs.readdir(dirPath);
  }
}

export { FileManager };
