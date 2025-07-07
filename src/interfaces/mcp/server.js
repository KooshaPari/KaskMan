/**
 * MCP (Model Context Protocol) Server for R&D Platform
 * Provides structured interface for Claude integration and AI-driven operations
 */

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ErrorCode,
  ListToolsRequestSchema,
  McpError,
} from '@modelcontextprotocol/sdk/types.js';
import { ProjectManager } from '../core/project-manager.js';
import { StatusMonitor } from '../core/status-monitor.js';
import { AuthManager } from '../core/auth-manager.js';
import { Logger } from '../core/logger.js';
import { AIOrchestrator } from '../core/ai-orchestrator.js';

class MCPServer {
  constructor(config = {}) {
    this.config = {
      name: 'rd-platform-mcp',
      version: '1.0.0',
      ...config,
    };

    this.server = new Server(
      {
        name: this.config.name,
        version: this.config.version,
      },
      {
        capabilities: {
          tools: {},
          resources: {},
          prompts: {},
        },
      }
    );

    this.projectManager = new ProjectManager();
    this.statusMonitor = new StatusMonitor();
    this.authManager = new AuthManager();
    this.aiOrchestrator = new AIOrchestrator();
    this.logger = new Logger('MCPServer');

    this.setupTools();
    this.setupResources();
    this.setupPrompts();
    this.setupHandlers();
  }

  setupTools() {
    // Project management tools
    this.server.setRequestHandler(ListToolsRequestSchema, async () => {
      return {
        tools: [
          {
            name: 'create_project',
            description: 'Create a new R&D project',
            inputSchema: {
              type: 'object',
              properties: {
                name: { type: 'string', description: 'Project name' },
                description: {
                  type: 'string',
                  description: 'Project description',
                },
                template: {
                  type: 'string',
                  description: 'Project template',
                  default: 'default',
                },
                private: {
                  type: 'boolean',
                  description: 'Make project private',
                  default: false,
                },
              },
              required: ['name'],
            },
          },
          {
            name: 'list_projects',
            description: 'List all projects',
            inputSchema: {
              type: 'object',
              properties: {
                includeArchived: {
                  type: 'boolean',
                  description: 'Include archived projects',
                  default: false,
                },
                filter: {
                  type: 'string',
                  description: 'Filter projects by name or tag',
                },
              },
            },
          },
          {
            name: 'get_project',
            description: 'Get detailed information about a project',
            inputSchema: {
              type: 'object',
              properties: {
                projectId: {
                  type: 'string',
                  description: 'Project ID or name',
                },
              },
              required: ['projectId'],
            },
          },
          {
            name: 'start_project',
            description: 'Start a project',
            inputSchema: {
              type: 'object',
              properties: {
                projectId: {
                  type: 'string',
                  description: 'Project ID or name',
                },
                options: {
                  type: 'object',
                  properties: {
                    port: { type: 'number', description: 'Port number' },
                    watch: {
                      type: 'boolean',
                      description: 'Watch for changes',
                    },
                    environment: {
                      type: 'string',
                      description: 'Environment (dev/staging/prod)',
                    },
                  },
                },
              },
              required: ['projectId'],
            },
          },
          {
            name: 'stop_project',
            description: 'Stop a project',
            inputSchema: {
              type: 'object',
              properties: {
                projectId: {
                  type: 'string',
                  description: 'Project ID or name',
                },
              },
              required: ['projectId'],
            },
          },
          {
            name: 'get_project_status',
            description: 'Get real-time project status',
            inputSchema: {
              type: 'object',
              properties: {
                projectId: {
                  type: 'string',
                  description: 'Project ID or name',
                },
              },
              required: ['projectId'],
            },
          },
          {
            name: 'get_project_logs',
            description: 'Get project logs',
            inputSchema: {
              type: 'object',
              properties: {
                projectId: {
                  type: 'string',
                  description: 'Project ID or name',
                },
                lines: {
                  type: 'number',
                  description: 'Number of lines to retrieve',
                  default: 100,
                },
                follow: {
                  type: 'boolean',
                  description: 'Follow logs in real-time',
                  default: false,
                },
              },
              required: ['projectId'],
            },
          },
          {
            name: 'update_project',
            description: 'Update project configuration',
            inputSchema: {
              type: 'object',
              properties: {
                projectId: {
                  type: 'string',
                  description: 'Project ID or name',
                },
                updates: {
                  type: 'object',
                  properties: {
                    name: { type: 'string' },
                    description: { type: 'string' },
                    tags: { type: 'array', items: { type: 'string' } },
                    config: { type: 'object' },
                  },
                },
              },
              required: ['projectId', 'updates'],
            },
          },
          {
            name: 'delete_project',
            description: 'Delete a project',
            inputSchema: {
              type: 'object',
              properties: {
                projectId: {
                  type: 'string',
                  description: 'Project ID or name',
                },
                force: {
                  type: 'boolean',
                  description: 'Force deletion without confirmation',
                  default: false,
                },
              },
              required: ['projectId'],
            },
          },
          {
            name: 'get_system_status',
            description: 'Get comprehensive system status',
            inputSchema: {
              type: 'object',
              properties: {
                detailed: {
                  type: 'boolean',
                  description: 'Include detailed metrics',
                  default: false,
                },
              },
            },
          },
          {
            name: 'get_system_health',
            description: 'Perform system health check',
            inputSchema: {
              type: 'object',
              properties: {
                includeServices: {
                  type: 'boolean',
                  description: 'Include service health',
                  default: true,
                },
              },
            },
          },
          {
            name: 'get_system_metrics',
            description: 'Get system performance metrics',
            inputSchema: {
              type: 'object',
              properties: {
                timeRange: {
                  type: 'string',
                  description: 'Time range (1h, 24h, 7d, 30d)',
                  default: '1h',
                },
                metrics: {
                  type: 'array',
                  items: { type: 'string' },
                  description: 'Specific metrics to retrieve',
                },
              },
            },
          },
          {
            name: 'trigger_maintenance',
            description: 'Trigger system maintenance tasks',
            inputSchema: {
              type: 'object',
              properties: {
                tasks: {
                  type: 'array',
                  items: { type: 'string' },
                  description: 'Maintenance tasks to run',
                },
                schedule: {
                  type: 'string',
                  description: 'Schedule maintenance (now, later, cron)',
                },
              },
            },
          },
          {
            name: 'orchestrate_ai_task',
            description: 'Orchestrate AI-driven task execution',
            inputSchema: {
              type: 'object',
              properties: {
                task: { type: 'string', description: 'Task description' },
                context: {
                  type: 'object',
                  description: 'Task context and parameters',
                },
                strategy: {
                  type: 'string',
                  description: 'Orchestration strategy',
                  default: 'auto',
                },
                agents: {
                  type: 'array',
                  items: { type: 'string' },
                  description: 'Specific agents to use',
                },
              },
              required: ['task'],
            },
          },
          {
            name: 'analyze_project_performance',
            description:
              'Analyze project performance and provide recommendations',
            inputSchema: {
              type: 'object',
              properties: {
                projectId: {
                  type: 'string',
                  description: 'Project ID or name',
                },
                analysisType: {
                  type: 'string',
                  description:
                    'Analysis type (performance, security, code-quality)',
                  default: 'performance',
                },
                timeRange: {
                  type: 'string',
                  description: 'Time range for analysis',
                  default: '24h',
                },
              },
              required: ['projectId'],
            },
          },
          {
            name: 'generate_project_report',
            description: 'Generate comprehensive project report',
            inputSchema: {
              type: 'object',
              properties: {
                projectId: {
                  type: 'string',
                  description: 'Project ID or name',
                },
                reportType: {
                  type: 'string',
                  description: 'Report type (summary, detailed, technical)',
                  default: 'summary',
                },
                format: {
                  type: 'string',
                  description: 'Output format (json, markdown, html)',
                  default: 'json',
                },
              },
              required: ['projectId'],
            },
          },
          {
            name: 'optimize_project_resources',
            description: 'Optimize project resource allocation',
            inputSchema: {
              type: 'object',
              properties: {
                projectId: {
                  type: 'string',
                  description: 'Project ID or name',
                },
                optimizationType: {
                  type: 'string',
                  description: 'Optimization type (cpu, memory, network)',
                  default: 'auto',
                },
                aggressive: {
                  type: 'boolean',
                  description: 'Use aggressive optimization',
                  default: false,
                },
              },
              required: ['projectId'],
            },
          },
        ],
      };
    });

    // Tool execution handler
    this.server.setRequestHandler(CallToolRequestSchema, async (request) => {
      const { name, arguments: args } = request.params;

      try {
        switch (name) {
          case 'create_project':
            return await this.handleCreateProject(args);
          case 'list_projects':
            return await this.handleListProjects(args);
          case 'get_project':
            return await this.handleGetProject(args);
          case 'start_project':
            return await this.handleStartProject(args);
          case 'stop_project':
            return await this.handleStopProject(args);
          case 'get_project_status':
            return await this.handleGetProjectStatus(args);
          case 'get_project_logs':
            return await this.handleGetProjectLogs(args);
          case 'update_project':
            return await this.handleUpdateProject(args);
          case 'delete_project':
            return await this.handleDeleteProject(args);
          case 'get_system_status':
            return await this.handleGetSystemStatus(args);
          case 'get_system_health':
            return await this.handleGetSystemHealth(args);
          case 'get_system_metrics':
            return await this.handleGetSystemMetrics(args);
          case 'trigger_maintenance':
            return await this.handleTriggerMaintenance(args);
          case 'orchestrate_ai_task':
            return await this.handleOrchestateAITask(args);
          case 'analyze_project_performance':
            return await this.handleAnalyzeProjectPerformance(args);
          case 'generate_project_report':
            return await this.handleGenerateProjectReport(args);
          case 'optimize_project_resources':
            return await this.handleOptimizeProjectResources(args);
          default:
            throw new McpError(
              ErrorCode.MethodNotFound,
              `Unknown tool: ${name}`
            );
        }
      } catch (error) {
        this.logger.error(`Tool execution error: ${name}`, error);
        throw new McpError(ErrorCode.InternalError, error.message);
      }
    });
  }

  setupResources() {
    // Resources provide access to platform data
    this.server.setRequestHandler('resources/list', async () => {
      return {
        resources: [
          {
            uri: 'rd-platform://projects',
            name: 'Projects',
            description: 'All projects in the platform',
            mimeType: 'application/json',
          },
          {
            uri: 'rd-platform://system/status',
            name: 'System Status',
            description: 'Real-time system status',
            mimeType: 'application/json',
          },
          {
            uri: 'rd-platform://metrics',
            name: 'System Metrics',
            description: 'Platform performance metrics',
            mimeType: 'application/json',
          },
        ],
      };
    });

    this.server.setRequestHandler('resources/read', async (request) => {
      const { uri } = request.params;

      try {
        switch (uri) {
          case 'rd-platform://projects': {
            const projects = await this.projectManager.listProjects();
            return {
              contents: [
                {
                  uri,
                  mimeType: 'application/json',
                  text: JSON.stringify(projects, null, 2),
                },
              ],
            };
          }
          case 'rd-platform://system/status': {
            const status = await this.statusMonitor.getSystemStatus();
            return {
              contents: [
                {
                  uri,
                  mimeType: 'application/json',
                  text: JSON.stringify(status, null, 2),
                },
              ],
            };
          }
          case 'rd-platform://metrics': {
            const metrics = await this.statusMonitor.getSystemMetrics();
            return {
              contents: [
                {
                  uri,
                  mimeType: 'application/json',
                  text: JSON.stringify(metrics, null, 2),
                },
              ],
            };
          }
          default:
            throw new McpError(
              ErrorCode.InvalidRequest,
              `Unknown resource: ${uri}`
            );
        }
      } catch (error) {
        this.logger.error(`Resource access error: ${uri}`, error);
        throw new McpError(ErrorCode.InternalError, error.message);
      }
    });
  }

  setupPrompts() {
    // Prompts provide context for AI interactions
    this.server.setRequestHandler('prompts/list', async () => {
      return {
        prompts: [
          {
            name: 'project_analysis',
            description: 'Analyze project structure and provide insights',
            arguments: [
              {
                name: 'projectId',
                description: 'Project to analyze',
                required: true,
              },
            ],
          },
          {
            name: 'system_optimization',
            description: 'Provide system optimization recommendations',
            arguments: [
              {
                name: 'focus',
                description: 'Optimization focus area',
                required: false,
              },
            ],
          },
          {
            name: 'troubleshooting',
            description: 'Help troubleshoot system issues',
            arguments: [
              {
                name: 'issue',
                description: 'Description of the issue',
                required: true,
              },
            ],
          },
        ],
      };
    });

    this.server.setRequestHandler('prompts/get', async (request) => {
      const { name, arguments: args } = request.params;

      try {
        switch (name) {
          case 'project_analysis':
            return await this.generateProjectAnalysisPrompt(args);
          case 'system_optimization':
            return await this.generateSystemOptimizationPrompt(args);
          case 'troubleshooting':
            return await this.generateTroubleshootingPrompt(args);
          default:
            throw new McpError(
              ErrorCode.InvalidRequest,
              `Unknown prompt: ${name}`
            );
        }
      } catch (error) {
        this.logger.error(`Prompt generation error: ${name}`, error);
        throw new McpError(ErrorCode.InternalError, error.message);
      }
    });
  }

  setupHandlers() {
    // Connection handlers
    this.server.onerror = (error) => {
      this.logger.error('MCP Server error:', error);
    };

    process.on('SIGINT', async () => {
      this.logger.info('Shutting down MCP server...');
      await this.server.close();
      process.exit(0);
    });
  }

  // Tool handlers
  async handleCreateProject(args) {
    const project = await this.projectManager.createProject(args);
    return {
      content: [
        {
          type: 'text',
          text: `Project "${project.name}" created successfully with ID: ${project.id}`,
        },
      ],
    };
  }

  async handleListProjects(args) {
    const projects = await this.projectManager.listProjects(args);
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(projects, null, 2),
        },
      ],
    };
  }

  async handleGetProject(args) {
    const project = await this.projectManager.getProject(args.projectId);
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(project, null, 2),
        },
      ],
    };
  }

  async handleStartProject(args) {
    const result = await this.projectManager.startProject(
      args.projectId,
      args.options
    );
    return {
      content: [
        {
          type: 'text',
          text: `Project started successfully. URL: ${result.url}, PID: ${result.pid}`,
        },
      ],
    };
  }

  async handleStopProject(args) {
    await this.projectManager.stopProject(args.projectId);
    return {
      content: [
        {
          type: 'text',
          text: `Project stopped successfully`,
        },
      ],
    };
  }

  async handleGetProjectStatus(args) {
    const status = await this.statusMonitor.getProjectStatus(args.projectId);
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(status, null, 2),
        },
      ],
    };
  }

  async handleGetProjectLogs(args) {
    const logs = await this.projectManager.getProjectLogs(args.projectId, args);
    return {
      content: [
        {
          type: 'text',
          text: logs.join('\n'),
        },
      ],
    };
  }

  async handleUpdateProject(args) {
    const project = await this.projectManager.updateProject(
      args.projectId,
      args.updates
    );
    return {
      content: [
        {
          type: 'text',
          text: `Project updated successfully`,
        },
      ],
    };
  }

  async handleDeleteProject(args) {
    await this.projectManager.deleteProject(args.projectId, args.force);
    return {
      content: [
        {
          type: 'text',
          text: `Project deleted successfully`,
        },
      ],
    };
  }

  async handleGetSystemStatus(args) {
    const status = await this.statusMonitor.getSystemStatus(args.detailed);
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(status, null, 2),
        },
      ],
    };
  }

  async handleGetSystemHealth(args) {
    const health = await this.statusMonitor.getHealthCheck(
      args.includeServices
    );
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(health, null, 2),
        },
      ],
    };
  }

  async handleGetSystemMetrics(args) {
    const metrics = await this.statusMonitor.getSystemMetrics(args);
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(metrics, null, 2),
        },
      ],
    };
  }

  async handleTriggerMaintenance(args) {
    const result = await this.statusMonitor.triggerMaintenance(args);
    return {
      content: [
        {
          type: 'text',
          text: `Maintenance triggered: ${result.tasks.join(', ')}`,
        },
      ],
    };
  }

  async handleOrchestateAITask(args) {
    const result = await this.aiOrchestrator.orchestrateTask(args);
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(result, null, 2),
        },
      ],
    };
  }

  async handleAnalyzeProjectPerformance(args) {
    const analysis = await this.aiOrchestrator.analyzeProjectPerformance(args);
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(analysis, null, 2),
        },
      ],
    };
  }

  async handleGenerateProjectReport(args) {
    const report = await this.aiOrchestrator.generateProjectReport(args);
    return {
      content: [
        {
          type: 'text',
          text:
            args.format === 'json' ? JSON.stringify(report, null, 2) : report,
        },
      ],
    };
  }

  async handleOptimizeProjectResources(args) {
    const optimization =
      await this.aiOrchestrator.optimizeProjectResources(args);
    return {
      content: [
        {
          type: 'text',
          text: JSON.stringify(optimization, null, 2),
        },
      ],
    };
  }

  // Prompt generators
  async generateProjectAnalysisPrompt(args) {
    const project = await this.projectManager.getProject(args.projectId);
    const status = await this.statusMonitor.getProjectStatus(args.projectId);

    return {
      description: `Analyze project ${project.name}`,
      messages: [
        {
          role: 'user',
          content: {
            type: 'text',
            text: `Please analyze this project and provide insights:

Project Information:
${JSON.stringify(project, null, 2)}

Current Status:
${JSON.stringify(status, null, 2)}

Please provide:
1. Project structure analysis
2. Performance insights
3. Potential improvements
4. Resource optimization suggestions
5. Security considerations`,
          },
        },
      ],
    };
  }

  async generateSystemOptimizationPrompt(args) {
    const systemStatus = await this.statusMonitor.getSystemStatus(true);
    const metrics = await this.statusMonitor.getSystemMetrics();

    return {
      description: 'System optimization recommendations',
      messages: [
        {
          role: 'user',
          content: {
            type: 'text',
            text: `Please analyze the system and provide optimization recommendations:

System Status:
${JSON.stringify(systemStatus, null, 2)}

Performance Metrics:
${JSON.stringify(metrics, null, 2)}

Focus Area: ${args.focus || 'overall'}

Please provide:
1. Performance bottlenecks
2. Resource optimization opportunities
3. Scalability recommendations
4. Security improvements
5. Maintenance suggestions`,
          },
        },
      ],
    };
  }

  async generateTroubleshootingPrompt(args) {
    const systemStatus = await this.statusMonitor.getSystemStatus(true);
    const health = await this.statusMonitor.getHealthCheck(true);

    return {
      description: 'Troubleshooting assistance',
      messages: [
        {
          role: 'user',
          content: {
            type: 'text',
            text: `Please help troubleshoot this issue:

Issue Description: ${args.issue}

System Status:
${JSON.stringify(systemStatus, null, 2)}

Health Check:
${JSON.stringify(health, null, 2)}

Please provide:
1. Potential root causes
2. Diagnostic steps
3. Resolution strategies
4. Prevention recommendations`,
          },
        },
      ],
    };
  }

  async start() {
    try {
      await this.projectManager.initialize();
      await this.statusMonitor.initialize();
      await this.aiOrchestrator.initialize();

      const transport = new StdioServerTransport();
      await this.server.connect(transport);

      this.logger.info('MCP Server started successfully');
      this.logger.info(`Server name: ${this.config.name}`);
      this.logger.info(`Server version: ${this.config.version}`);

      return {
        name: this.config.name,
        version: this.config.version,
        transport: 'stdio',
      };
    } catch (error) {
      this.logger.error('Failed to start MCP server:', error);
      throw error;
    }
  }

  async stop() {
    try {
      await this.server.close();
      await this.statusMonitor.stop();
      await this.aiOrchestrator.stop();
      this.logger.info('MCP Server stopped successfully');
    } catch (error) {
      this.logger.error('Error stopping MCP server:', error);
      throw error;
    }
  }
}

export { MCPServer };

// CLI integration
if (import.meta.url === `file://${process.argv[1]}`) {
  const mcpServer = new MCPServer();

  mcpServer.start().catch((error) => {
    console.error('Failed to start MCP server:', error);
    process.exit(1);
  });
}
