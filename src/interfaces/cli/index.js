#!/usr/bin/env node

/**
 * CLI Interface for R&D Platform
 * Provides command-line interface for project management and system control
 */

import { program } from 'commander';
import chalk from 'chalk';
import inquirer from 'inquirer';
import { ProjectManager } from '../core/project-manager.js';
import { StatusMonitor } from '../core/status-monitor.js';
import { AuthManager } from '../core/auth-manager.js';
import { APIClient } from '../core/api-client.js';

const VERSION = '1.0.0';
const projectManager = new ProjectManager();
const statusMonitor = new StatusMonitor();
const authManager = new AuthManager();
const apiClient = new APIClient();

// Global error handler
process.on('uncaughtException', (error) => {
  console.error(chalk.red('✖ Uncaught Exception:'), error.message);
  process.exit(1);
});

process.on('unhandledRejection', (reason) => {
  console.error(chalk.red('✖ Unhandled Rejection:'), reason);
  process.exit(1);
});

// Setup CLI
program
  .name('rd-platform')
  .description('R&D Platform CLI - Project Management and System Control')
  .version(VERSION);

// Authentication commands
program
  .command('auth')
  .description('Authentication management')
  .addCommand(
    program
      .createCommand('login')
      .description('Login to R&D Platform')
      .option('-u, --username <username>', 'Username')
      .option('-p, --password <password>', 'Password')
      .option('--token <token>', 'API token')
      .action(async (options) => {
        try {
          let credentials = options;

          if (!credentials.username && !credentials.token) {
            credentials = await inquirer.prompt([
              {
                type: 'input',
                name: 'username',
                message: 'Username:',
                validate: (input) => input.length > 0 || 'Username is required',
              },
              {
                type: 'password',
                name: 'password',
                message: 'Password:',
                mask: '*',
                validate: (input) => input.length > 0 || 'Password is required',
              },
            ]);
          }

          const result = await authManager.login(credentials);
          console.log(chalk.green('✓ Successfully logged in'));
          console.log(chalk.dim(`Token: ${result.token}`));
        } catch (error) {
          console.error(chalk.red('✖ Login failed:'), error.message);
          process.exit(1);
        }
      })
  )
  .addCommand(
    program
      .createCommand('logout')
      .description('Logout from R&D Platform')
      .action(async () => {
        try {
          await authManager.logout();
          console.log(chalk.green('✓ Successfully logged out'));
        } catch (error) {
          console.error(chalk.red('✖ Logout failed:'), error.message);
        }
      })
  )
  .addCommand(
    program
      .createCommand('status')
      .description('Check authentication status')
      .action(async () => {
        try {
          const status = await authManager.getStatus();
          if (status.authenticated) {
            console.log(chalk.green('✓ Authenticated'));
            console.log(chalk.dim(`User: ${status.user.username}`));
            console.log(chalk.dim(`Role: ${status.user.role}`));
            console.log(
              chalk.dim(
                `Expires: ${new Date(status.expiresAt).toLocaleString()}`
              )
            );
          } else {
            console.log(chalk.yellow('⚠ Not authenticated'));
          }
        } catch (error) {
          console.error(chalk.red('✖ Status check failed:'), error.message);
        }
      })
  );

// Project management commands
program
  .command('project')
  .description('Project management')
  .addCommand(
    program
      .createCommand('create')
      .description('Create new project')
      .argument('<name>', 'Project name')
      .option('-d, --description <description>', 'Project description')
      .option('-t, --template <template>', 'Project template')
      .option('--private', 'Make project private')
      .action(async (name, options) => {
        try {
          await authManager.requireAuth();

          const projectConfig = {
            name,
            description: options.description || '',
            template: options.template || 'default',
            private: options.private || false,
          };

          const project = await projectManager.createProject(projectConfig);
          console.log(chalk.green('✓ Project created successfully'));
          console.log(chalk.dim(`ID: ${project.id}`));
          console.log(chalk.dim(`Path: ${project.path}`));
        } catch (error) {
          console.error(
            chalk.red('✖ Project creation failed:'),
            error.message
          );
          process.exit(1);
        }
      })
  )
  .addCommand(
    program
      .createCommand('list')
      .description('List projects')
      .option('-a, --all', 'Show all projects (including archived)')
      .option('-f, --format <format>', 'Output format (table, json)', 'table')
      .action(async (options) => {
        try {
          await authManager.requireAuth();

          const projects = await projectManager.listProjects({
            includeArchived: options.all,
          });

          if (options.format === 'json') {
            console.log(JSON.stringify(projects, null, 2));
          } else {
            console.table(
              projects.map((p) => ({
                ID: p.id,
                Name: p.name,
                Status: p.status,
                Created: new Date(p.createdAt).toLocaleDateString(),
                'Last Modified': new Date(p.updatedAt).toLocaleDateString(),
              }))
            );
          }
        } catch (error) {
          console.error(
            chalk.red('✖ Failed to list projects:'),
            error.message
          );
        }
      })
  )
  .addCommand(
    program
      .createCommand('start')
      .description('Start project')
      .argument('<project-id>', 'Project ID or name')
      .option('-w, --watch', 'Watch for changes')
      .option('-p, --port <port>', 'Port number', '3000')
      .action(async (projectId, options) => {
        try {
          await authManager.requireAuth();

          const result = await projectManager.startProject(projectId, {
            watch: options.watch,
            port: parseInt(options.port),
          });

          console.log(chalk.green('✓ Project started successfully'));
          console.log(chalk.dim(`URL: ${result.url}`));
          console.log(chalk.dim(`PID: ${result.pid}`));

          if (options.watch) {
            console.log(chalk.blue('👁 Watching for changes...'));
            // Keep process alive for watching
            process.stdin.setRawMode(true);
            process.stdin.resume();
            process.stdin.on('data', (key) => {
              if (key.toString() === '\u0003') {
                // Ctrl+C
                console.log(chalk.yellow('\n⚠ Stopping project...'));
                projectManager.stopProject(projectId);
                process.exit(0);
              }
            });
          }
        } catch (error) {
          console.error(
            chalk.red('✖ Failed to start project:'),
            error.message
          );
          process.exit(1);
        }
      })
  )
  .addCommand(
    program
      .createCommand('stop')
      .description('Stop project')
      .argument('<project-id>', 'Project ID or name')
      .action(async (projectId) => {
        try {
          await authManager.requireAuth();
          await projectManager.stopProject(projectId);
          console.log(chalk.green('✓ Project stopped successfully'));
        } catch (error) {
          console.error(chalk.red('✖ Failed to stop project:'), error.message);
        }
      })
  )
  .addCommand(
    program
      .createCommand('status')
      .description('Show project status')
      .argument('[project-id]', 'Project ID or name (optional)')
      .option('-w, --watch', 'Watch status updates')
      .action(async (projectId, options) => {
        try {
          await authManager.requireAuth();

          if (options.watch) {
            console.log(chalk.blue('👁 Watching project status...'));
            const watcher = statusMonitor.watchProject(projectId);

            watcher.on('status', (status) => {
              console.clear();
              displayProjectStatus(status);
            });

            watcher.on('error', (error) => {
              console.error(
                chalk.red('✖ Status monitoring error:'),
                error.message
              );
            });

            // Keep process alive
            process.stdin.setRawMode(true);
            process.stdin.resume();
            process.stdin.on('data', (key) => {
              if (key.toString() === '\u0003') {
                // Ctrl+C
                console.log(chalk.yellow('\n⚠ Stopping status monitoring...'));
                watcher.stop();
                process.exit(0);
              }
            });
          } else {
            const status = await statusMonitor.getProjectStatus(projectId);
            displayProjectStatus(status);
          }
        } catch (error) {
          console.error(
            chalk.red('✖ Failed to get project status:'),
            error.message
          );
        }
      })
  );

// System commands
program
  .command('system')
  .description('System management')
  .addCommand(
    program
      .createCommand('status')
      .description('Show system status')
      .option('-w, --watch', 'Watch system status')
      .action(async (options) => {
        try {
          if (options.watch) {
            console.log(chalk.blue('👁 Watching system status...'));
            const watcher = statusMonitor.watchSystem();

            watcher.on('status', (status) => {
              console.clear();
              displaySystemStatus(status);
            });

            // Keep process alive
            process.stdin.setRawMode(true);
            process.stdin.resume();
            process.stdin.on('data', (key) => {
              if (key.toString() === '\u0003') {
                // Ctrl+C
                console.log(chalk.yellow('\n⚠ Stopping system monitoring...'));
                watcher.stop();
                process.exit(0);
              }
            });
          } else {
            const status = await statusMonitor.getSystemStatus();
            displaySystemStatus(status);
          }
        } catch (error) {
          console.error(
            chalk.red('✖ Failed to get system status:'),
            error.message
          );
        }
      })
  )
  .addCommand(
    program
      .createCommand('health')
      .description('System health check')
      .action(async () => {
        try {
          const health = await statusMonitor.getHealthCheck();

          console.log(chalk.bold('System Health Check'));
          console.log('═'.repeat(40));

          for (const [service, status] of Object.entries(health.services)) {
            const icon = status.healthy ? '✓' : '✖';
            const color = status.healthy ? 'green' : 'red';
            console.log(chalk[color](`${icon} ${service}: ${status.status}`));

            if (status.details) {
              console.log(chalk.dim(`  ${status.details}`));
            }
          }

          console.log('═'.repeat(40));
          const overallColor = health.overall.healthy ? 'green' : 'red';
          console.log(chalk[overallColor](`Overall: ${health.overall.status}`));

          if (!health.overall.healthy) {
            process.exit(1);
          }
        } catch (error) {
          console.error(chalk.red('✖ Health check failed:'), error.message);
          process.exit(1);
        }
      })
  );

// Server commands
program
  .command('server')
  .description('Server management')
  .addCommand(
    program
      .createCommand('start')
      .description('Start API server')
      .option('-p, --port <port>', 'Port number', '8080')
      .option('-h, --host <host>', 'Host address', 'localhost')
      .option('-d, --daemon', 'Run as daemon')
      .action(async (options) => {
        try {
          await authManager.requireAuth();

          const serverConfig = {
            port: parseInt(options.port),
            host: options.host,
            daemon: options.daemon,
          };

          const server = await apiClient.startServer(serverConfig);
          console.log(chalk.green('✓ API server started successfully'));
          console.log(chalk.dim(`URL: ${server.url}`));
          console.log(chalk.dim(`PID: ${server.pid}`));

          if (!options.daemon) {
            console.log(chalk.blue('Press Ctrl+C to stop the server'));
            process.on('SIGINT', () => {
              console.log(chalk.yellow('\n⚠ Stopping server...'));
              apiClient.stopServer();
              process.exit(0);
            });
          }
        } catch (error) {
          console.error(chalk.red('✖ Failed to start server:'), error.message);
          process.exit(1);
        }
      })
  )
  .addCommand(
    program
      .createCommand('stop')
      .description('Stop API server')
      .action(async () => {
        try {
          await apiClient.stopServer();
          console.log(chalk.green('✓ API server stopped successfully'));
        } catch (error) {
          console.error(chalk.red('✖ Failed to stop server:'), error.message);
        }
      })
  );

// Helper functions
function displayProjectStatus(status) {
  console.log(chalk.bold(`Project Status: ${status.project.name}`));
  console.log('═'.repeat(50));
  console.log(`ID: ${status.project.id}`);
  console.log(
    `Status: ${getStatusColor(status.project.status)(status.project.status)}`
  );
  console.log(`CPU: ${status.resources.cpu}%`);
  console.log(`Memory: ${status.resources.memory}%`);
  console.log(`Uptime: ${status.uptime}`);
  console.log(`Last Updated: ${new Date(status.updatedAt).toLocaleString()}`);

  if (status.processes && status.processes.length > 0) {
    console.log('\nProcesses:');
    status.processes.forEach((proc) => {
      console.log(
        `  ${proc.name} (${proc.pid}): ${getStatusColor(proc.status)(proc.status)}`
      );
    });
  }
}

function displaySystemStatus(status) {
  console.log(chalk.bold('System Status'));
  console.log('═'.repeat(30));
  console.log(`CPU: ${status.cpu}%`);
  console.log(`Memory: ${status.memory}%`);
  console.log(`Disk: ${status.disk}%`);
  console.log(`Active Projects: ${status.activeProjects}`);
  console.log(`Uptime: ${status.uptime}`);
  console.log(`Load Average: ${status.loadAverage.join(', ')}`);
}

function getStatusColor(status) {
  switch (status.toLowerCase()) {
    case 'running':
    case 'active':
    case 'healthy':
      return chalk.green;
    case 'stopped':
    case 'inactive':
      return chalk.red;
    case 'starting':
    case 'stopping':
    case 'pending':
      return chalk.yellow;
    default:
      return chalk.dim;
  }
}

// Parse command line arguments
program.parse();

// Show help if no command provided
if (!process.argv.slice(2).length) {
  program.outputHelp();
}
