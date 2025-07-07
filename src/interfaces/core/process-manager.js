/**
 * Process Manager
 * Handles process lifecycle management and monitoring
 */

import { EventEmitter } from 'events';
import { spawn, exec } from 'child_process';
import { promisify } from 'util';
import { promises as fs } from 'fs';
import path from 'path';
import { Logger } from './logger.js';

const execAsync = promisify(exec);

class ProcessManager extends EventEmitter {
  constructor(config = {}) {
    super();
    this.config = {
      processesFile: config.processesFile || './data/processes.json',
      logsDir: config.logsDir || './logs',
      maxLogSize: config.maxLogSize || 10 * 1024 * 1024, // 10MB
      processTimeout: config.processTimeout || 30000,
      ...config,
    };

    this.logger = new Logger('ProcessManager');
    this.processes = new Map();
    this.processLogs = new Map();
    this.initialized = false;
  }

  async initialize() {
    try {
      await this.ensureDirectories();
      await this.loadProcesses();
      await this.recoverProcesses();

      this.initialized = true;
      this.logger.info('ProcessManager initialized successfully');
    } catch (error) {
      this.logger.error('Failed to initialize ProcessManager:', error);
      throw error;
    }
  }

  async ensureDirectories() {
    const dirs = [path.dirname(this.config.processesFile), this.config.logsDir];

    for (const dir of dirs) {
      await fs.mkdir(dir, { recursive: true });
    }
  }

  async loadProcesses() {
    try {
      const processesData = await fs.readFile(
        this.config.processesFile,
        'utf8'
      );
      const processes = JSON.parse(processesData);

      for (const processInfo of processes) {
        this.processes.set(processInfo.id, processInfo);
      }

      this.logger.info(`Loaded ${this.processes.size} processes`);
    } catch (error) {
      if (error.code === 'ENOENT') {
        this.logger.info(
          'No processes file found, starting with empty process database'
        );
        await this.saveProcesses();
      } else {
        this.logger.error('Failed to load processes:', error);
        throw error;
      }
    }
  }

  async saveProcesses() {
    try {
      const processes = Array.from(this.processes.values());
      await fs.writeFile(
        this.config.processesFile,
        JSON.stringify(processes, null, 2)
      );
    } catch (error) {
      this.logger.error('Failed to save processes:', error);
      throw error;
    }
  }

  async recoverProcesses() {
    const runningProcesses = [];

    for (const [id, processInfo] of this.processes.entries()) {
      if (processInfo.status === 'running' && processInfo.pid) {
        try {
          // Check if process is still running
          process.kill(processInfo.pid, 0);
          runningProcesses.push(id);
          this.logger.info(
            `Recovered running process: ${id} (PID: ${processInfo.pid})`
          );
        } catch (error) {
          // Process is not running anymore
          processInfo.status = 'stopped';
          processInfo.exitCode = -1;
          processInfo.stoppedAt = new Date().toISOString();
          this.logger.info(
            `Process ${id} is no longer running, marked as stopped`
          );
        }
      }
    }

    if (runningProcesses.length > 0) {
      await this.saveProcesses();
    }
  }

  async startProject(projectId, workingDir, command, options = {}) {
    try {
      const processId = `project-${projectId}`;

      // Check if project is already running
      if (this.processes.has(processId)) {
        const existing = this.processes.get(processId);
        if (existing.status === 'running') {
          throw new Error(`Project ${projectId} is already running`);
        }
      }

      // Parse command
      const commandParts = command.split(' ');
      const cmd = commandParts[0];
      const args = commandParts.slice(1);

      // Setup environment
      const env = {
        ...process.env,
        NODE_ENV: options.environment || 'development',
        PORT: options.port || 3000,
        ...options.env,
      };

      // Create log files
      const logDir = path.join(this.config.logsDir, projectId);
      await fs.mkdir(logDir, { recursive: true });

      const stdoutLogPath = path.join(logDir, 'stdout.log');
      const stderrLogPath = path.join(logDir, 'stderr.log');

      // Start process
      const childProcess = spawn(cmd, args, {
        cwd: workingDir,
        env,
        stdio: ['ignore', 'pipe', 'pipe'],
      });

      // Setup log streams
      const stdoutStream = await fs.open(stdoutLogPath, 'a');
      const stderrStream = await fs.open(stderrLogPath, 'a');

      childProcess.stdout.on('data', (data) => {
        stdoutStream.write(data);
        this.emit('process:stdout', processId, data.toString());
      });

      childProcess.stderr.on('data', (data) => {
        stderrStream.write(data);
        this.emit('process:stderr', processId, data.toString());
      });

      // Process event handlers
      childProcess.on('error', (error) => {
        this.logger.error(`Process ${processId} error:`, error);
        this.emit('process:error', processId, error);
      });

      childProcess.on('exit', (code, signal) => {
        this.logger.info(
          `Process ${processId} exited with code ${code} and signal ${signal}`
        );

        // Update process info
        const processInfo = this.processes.get(processId);
        if (processInfo) {
          processInfo.status = 'stopped';
          processInfo.exitCode = code;
          processInfo.signal = signal;
          processInfo.stoppedAt = new Date().toISOString();
          this.saveProcesses();
        }

        // Close log streams
        stdoutStream.close();
        stderrStream.close();

        this.emit('process:exit', processId, code, signal);
      });

      // Create process info
      const processInfo = {
        id: processId,
        projectId,
        pid: childProcess.pid,
        command,
        workingDir,
        status: 'running',
        startTime: Date.now(),
        startedAt: new Date().toISOString(),
        port: options.port || 3000,
        url: `http://localhost:${options.port || 3000}`,
        logFiles: {
          stdout: stdoutLogPath,
          stderr: stderrLogPath,
        },
        options,
      };

      this.processes.set(processId, processInfo);
      await this.saveProcesses();

      this.emit('process:started', processId, processInfo);
      this.logger.info(
        `Started process ${processId} (PID: ${childProcess.pid})`
      );

      return processInfo;
    } catch (error) {
      this.logger.error(`Failed to start project ${projectId}:`, error);
      throw error;
    }
  }

  async stopProject(projectId) {
    try {
      const processId = `project-${projectId}`;
      const processInfo = this.processes.get(processId);

      if (!processInfo) {
        throw new Error(`Process not found: ${processId}`);
      }

      if (processInfo.status !== 'running') {
        throw new Error(`Process is not running: ${processId}`);
      }

      // Send SIGTERM for graceful shutdown
      process.kill(processInfo.pid, 'SIGTERM');

      // Wait for graceful shutdown
      await new Promise((resolve) => {
        const timeout = setTimeout(() => {
          this.logger.warn(`Forcing shutdown of process ${processId}`);
          try {
            process.kill(processInfo.pid, 'SIGKILL');
          } catch (error) {
            // Process might already be dead
          }
          resolve();
        }, 10000);

        const checkInterval = setInterval(() => {
          try {
            process.kill(processInfo.pid, 0);
          } catch (error) {
            // Process is dead
            clearTimeout(timeout);
            clearInterval(checkInterval);
            resolve();
          }
        }, 100);
      });

      // Update process info
      processInfo.status = 'stopped';
      processInfo.stoppedAt = new Date().toISOString();
      await this.saveProcesses();

      this.emit('process:stopped', processId);
      this.logger.info(`Stopped process ${processId}`);

      return processInfo;
    } catch (error) {
      this.logger.error(`Failed to stop project ${projectId}:`, error);
      throw error;
    }
  }

  async getProjectLogs(projectId, options = {}) {
    try {
      const processId = `project-${projectId}`;
      const processInfo = this.processes.get(processId);

      if (!processInfo) {
        throw new Error(`Process not found: ${processId}`);
      }

      const logType = options.stderr ? 'stderr' : 'stdout';
      const logFile = processInfo.logFiles[logType];

      if (!logFile) {
        return [];
      }

      const logContent = await fs.readFile(logFile, 'utf8');
      const lines = logContent.split('\n').filter((line) => line.trim());

      // Return last N lines
      const numLines = options.lines || 100;
      return lines.slice(-numLines);
    } catch (error) {
      this.logger.error(`Failed to get logs for project ${projectId}:`, error);
      throw error;
    }
  }

  async runCommand(workingDir, command, options = {}) {
    try {
      const result = await execAsync(command, {
        cwd: workingDir,
        timeout: options.timeout || this.config.processTimeout,
        env: { ...process.env, ...options.env },
      });

      return {
        stdout: result.stdout,
        stderr: result.stderr,
        exitCode: 0,
      };
    } catch (error) {
      return {
        stdout: error.stdout || '',
        stderr: error.stderr || '',
        exitCode: error.code || 1,
        error: error.message,
      };
    }
  }

  getRunningProjects() {
    const runningProjects = new Map();

    for (const [_id, processInfo] of this.processes.entries()) {
      if (processInfo.status === 'running' && processInfo.projectId) {
        runningProjects.set(processInfo.projectId, processInfo);
      }
    }

    return runningProjects;
  }

  getActiveProjectsCount() {
    return Array.from(this.processes.values()).filter(
      (p) => p.status === 'running' && p.projectId
    ).length;
  }

  async getProcessCount() {
    try {
      const result = await execAsync('ps aux | wc -l');
      return parseInt(result.stdout.trim()) || 0;
    } catch (error) {
      return 0;
    }
  }

  isHealthy() {
    // Check if process manager is functioning correctly
    return this.initialized && this.processes.size >= 0;
  }

  async cleanupLogs() {
    try {
      const logDirs = await fs.readdir(this.config.logsDir);

      for (const logDir of logDirs) {
        const logPath = path.join(this.config.logsDir, logDir);
        const stats = await fs.stat(logPath);

        if (stats.isDirectory()) {
          const files = await fs.readdir(logPath);

          for (const file of files) {
            const filePath = path.join(logPath, file);
            const fileStats = await fs.stat(filePath);

            // Remove logs older than 30 days
            if (
              Date.now() - fileStats.mtime.getTime() >
              30 * 24 * 60 * 60 * 1000
            ) {
              await fs.unlink(filePath);
              this.logger.info(`Cleaned up old log file: ${filePath}`);
            }
          }
        }
      }
    } catch (error) {
      this.logger.error('Failed to cleanup logs:', error);
    }
  }

  async stop() {
    try {
      // Stop all running processes
      const runningProcesses = Array.from(this.processes.values()).filter(
        (p) => p.status === 'running'
      );

      for (const processInfo of runningProcesses) {
        if (processInfo.projectId) {
          await this.stopProject(processInfo.projectId);
        }
      }

      await this.saveProcesses();

      this.logger.info('ProcessManager stopped successfully');
    } catch (error) {
      this.logger.error('Error stopping ProcessManager:', error);
      throw error;
    }
  }
}

export { ProcessManager };
