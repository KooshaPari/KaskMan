/**
 * Project Manager
 * Handles project lifecycle management, creation, deployment, and operations
 */

import { EventEmitter } from 'events';
import { promises as fs } from 'fs';
import path from 'path';
import { v4 as uuidv4 } from 'uuid';
import { Logger } from './logger.js';
import { ConfigManager } from './config-manager.js';
import { FileManager } from './file-manager.js';
import { ProcessManager } from './process-manager.js';

class ProjectManager extends EventEmitter {
  constructor(config = {}) {
    super();
    this.config = {
      projectsDir: config.projectsDir || './projects',
      templatesDir: config.templatesDir || './templates',
      defaultTemplate: config.defaultTemplate || 'default',
      ...config,
    };

    this.logger = new Logger('ProjectManager');
    this.configManager = new ConfigManager();
    this.fileManager = new FileManager();
    this.processManager = new ProcessManager();

    this.projects = new Map();
    this.runningProjects = new Map();
    this.templates = new Map();

    // Add default template immediately for testing
    this.addDefaultTemplate();
  }

  addDefaultTemplate() {
    const defaultTemplate = {
      name: 'default',
      description: 'Default R&D project template',
      structure: {
        'src/': {},
        'tests/': {},
        'docs/': {},
        'config/': {},
        'package.json': {
          name: '{{projectName}}',
          version: '1.0.0',
          description: '{{projectDescription}}',
          main: 'src/index.js',
          scripts: {
            start: 'node src/index.js',
            test: 'npm test',
            dev: 'nodemon src/index.js',
          },
          dependencies: {},
          devDependencies: {},
        },
        'README.md':
          '# {{projectName}}\n\n{{projectDescription}}\n\n## Getting Started\n\n```bash\nnpm install\nnpm start\n```',
        '.gitignore': 'node_modules/\n*.log\n.env\n.DS_Store',
      },
      commands: {
        install: 'npm install',
        start: 'npm start',
        test: 'npm test',
        build: 'npm run build',
      },
    };

    this.templates.set('default', defaultTemplate);
  }

  async initialize() {
    try {
      await this.ensureDirectories();
      await this.loadTemplates();
      await this.loadProjects();
      await this.recoverRunningProjects();

      this.logger.info('ProjectManager initialized successfully');
    } catch (error) {
      this.logger.error('Failed to initialize ProjectManager:', error);
      throw error;
    }
  }

  async ensureDirectories() {
    const dirs = [this.config.projectsDir, this.config.templatesDir];

    for (const dir of dirs) {
      await fs.mkdir(dir, { recursive: true });
    }
  }

  async loadTemplates() {
    try {
      const templateFiles = await fs.readdir(this.config.templatesDir);

      for (const templateFile of templateFiles) {
        if (templateFile.endsWith('.json')) {
          const templatePath = path.join(
            this.config.templatesDir,
            templateFile
          );
          const templateData = await fs.readFile(templatePath, 'utf8');
          const template = JSON.parse(templateData);

          this.templates.set(template.name, template);
        }
      }

      // Create default template if none exists
      if (!this.templates.has('default')) {
        const defaultTemplate = {
          name: 'default',
          description: 'Default R&D project template',
          structure: {
            'src/': {},
            'tests/': {},
            'docs/': {},
            'config/': {},
            'package.json': {
              name: '{{projectName}}',
              version: '1.0.0',
              description: '{{projectDescription}}',
              main: 'src/index.js',
              scripts: {
                start: 'node src/index.js',
                test: 'npm test',
                dev: 'nodemon src/index.js',
              },
              dependencies: {},
              devDependencies: {},
            },
            'README.md':
              '# {{projectName}}\n\n{{projectDescription}}\n\n## Getting Started\n\n```bash\nnpm install\nnpm start\n```',
            '.gitignore': 'node_modules/\n*.log\n.env\n.DS_Store',
          },
          commands: {
            install: 'npm install',
            start: 'npm start',
            test: 'npm test',
            build: 'npm run build',
          },
        };

        this.templates.set('default', defaultTemplate);

        // Save default template
        const templatePath = path.join(
          this.config.templatesDir,
          'default.json'
        );
        await fs.writeFile(
          templatePath,
          JSON.stringify(defaultTemplate, null, 2)
        );
      }

      this.logger.info(`Loaded ${this.templates.size} project templates`);
    } catch (error) {
      this.logger.error('Failed to load templates:', error);
      throw error;
    }
  }

  async loadProjects() {
    try {
      const projectDirs = await fs.readdir(this.config.projectsDir);

      for (const projectDir of projectDirs) {
        const projectPath = path.join(this.config.projectsDir, projectDir);
        const configPath = path.join(projectPath, 'project.json');

        try {
          const configData = await fs.readFile(configPath, 'utf8');
          const project = JSON.parse(configData);

          project.path = projectPath;
          project.configPath = configPath;

          this.projects.set(project.id, project);
        } catch (error) {
          this.logger.warn(
            `Failed to load project ${projectDir}:`,
            error.message
          );
        }
      }

      this.logger.info(`Loaded ${this.projects.size} projects`);
    } catch (error) {
      this.logger.error('Failed to load projects:', error);
      throw error;
    }
  }

  async recoverRunningProjects() {
    try {
      const runningProjects = await this.processManager.getRunningProjects();

      for (const [projectId, processInfo] of runningProjects) {
        if (this.projects.has(projectId)) {
          this.runningProjects.set(projectId, processInfo);
          this.logger.info(`Recovered running project: ${projectId}`);
        }
      }
    } catch (error) {
      this.logger.error('Failed to recover running projects:', error);
    }
  }

  async createProject(config) {
    try {
      // Validate project name
      if (!config.name || config.name.trim() === '') {
        throw new Error('Project name is required');
      }

      const projectId = uuidv4();
      const projectName = config.name.replace(/[^a-zA-Z0-9-_]/g, '-');
      const projectPath = path.join(this.config.projectsDir, projectName);

      // Check if project already exists
      if (await this.fileManager.exists(projectPath)) {
        throw new Error('Project name already exists');
      }

      // Get template
      const templateName = config.template || this.config.defaultTemplate;
      const template = this.templates.get(templateName);

      if (!template) {
        throw new Error(`Template not found: ${templateName}`);
      }

      // Create project structure
      await fs.mkdir(projectPath, { recursive: true });
      await this.createProjectFromTemplate(projectPath, template, config);

      // Create project configuration
      const project = {
        id: projectId,
        name: config.name,
        description: config.description || '',
        template: templateName,
        private: config.private || false,
        path: projectPath,
        configPath: path.join(projectPath, 'project.json'),
        status: 'created',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        tags: config.tags || [],
        metadata: config.metadata || {},
      };

      // Save project configuration
      await fs.writeFile(project.configPath, JSON.stringify(project, null, 2));

      // Add to projects map
      this.projects.set(projectId, project);

      this.emit('project:created', project);
      this.logger.info(`Project created: ${project.name} (${projectId})`);

      return project;
    } catch (error) {
      this.logger.error('Failed to create project:', error);
      throw error;
    }
  }

  async createProjectFromTemplate(projectPath, template, config) {
    const replacements = {
      '{{projectName}}': config.name,
      '{{projectDescription}}': config.description || '',
      '{{projectId}}': uuidv4(),
      '{{timestamp}}': new Date().toISOString(),
    };

    await this.createStructureRecursive(
      projectPath,
      template.structure,
      replacements
    );
  }

  async createStructureRecursive(basePath, structure, replacements) {
    for (const [name, content] of Object.entries(structure)) {
      const fullPath = path.join(basePath, name);

      if (name.endsWith('/')) {
        // Directory
        await fs.mkdir(fullPath, { recursive: true });
        if (typeof content === 'object') {
          await this.createStructureRecursive(fullPath, content, replacements);
        }
      } else {
        // File
        let fileContent = '';

        if (typeof content === 'string') {
          fileContent = content;
        } else if (typeof content === 'object') {
          fileContent = JSON.stringify(content, null, 2);
        }

        // Replace placeholders
        for (const [placeholder, value] of Object.entries(replacements)) {
          fileContent = fileContent.replace(
            new RegExp(placeholder, 'g'),
            value
          );
        }

        await fs.writeFile(fullPath, fileContent);
      }
    }
  }

  async listProjects(options = {}) {
    try {
      const projects = Array.from(this.projects.values());

      let filteredProjects = projects;

      // Filter archived projects
      if (!options.includeArchived) {
        filteredProjects = filteredProjects.filter(
          (p) => p.status !== 'archived'
        );
      }

      // Apply text filter
      if (options.filter) {
        const filterLower = options.filter.toLowerCase();
        filteredProjects = filteredProjects.filter(
          (p) =>
            p.name.toLowerCase().includes(filterLower) ||
            p.description.toLowerCase().includes(filterLower) ||
            p.tags.some((tag) => tag.toLowerCase().includes(filterLower))
        );
      }

      return filteredProjects.sort(
        (a, b) => new Date(b.updatedAt) - new Date(a.updatedAt)
      );
    } catch (error) {
      this.logger.error('Failed to list projects:', error);
      throw error;
    }
  }

  async getProject(projectId) {
    try {
      // Try to find by ID first
      let project = this.projects.get(projectId);

      // If not found, try to find by name
      if (!project) {
        project = Array.from(this.projects.values()).find(
          (p) => p.name === projectId
        );
      }

      if (!project) {
        throw new Error(`Project not found: ${projectId}`);
      }

      return project;
    } catch (error) {
      this.logger.error('Failed to get project:', error);
      throw error;
    }
  }

  async updateProject(projectId, updates) {
    try {
      const project = await this.getProject(projectId);

      // Update project configuration
      const updatedProject = {
        ...project,
        ...updates,
        updatedAt: new Date().toISOString(),
      };

      // Save updated configuration
      await fs.writeFile(
        project.configPath,
        JSON.stringify(updatedProject, null, 2)
      );

      // Update in memory
      this.projects.set(project.id, updatedProject);

      this.emit('project:updated', updatedProject);
      this.logger.info(`Project updated: ${project.name} (${project.id})`);

      return updatedProject;
    } catch (error) {
      this.logger.error('Failed to update project:', error);
      throw error;
    }
  }

  async deleteProject(projectId, force = false) {
    try {
      const project = await this.getProject(projectId);

      // Stop project if running
      if (this.runningProjects.has(project.id)) {
        await this.stopProject(project.id);
      }

      // Remove project files
      if (force) {
        await this.fileManager.removeDirectory(project.path);
      } else {
        // Archive project instead of deleting
        await this.updateProject(project.id, { status: 'archived' });
      }

      // Remove from memory if force delete
      if (force) {
        this.projects.delete(project.id);
      }

      this.emit('project:deleted', project);
      this.logger.info(
        `Project ${force ? 'deleted' : 'archived'}: ${project.name} (${project.id})`
      );

      return project;
    } catch (error) {
      this.logger.error('Failed to delete project:', error);
      throw error;
    }
  }

  async startProject(projectId, options = {}) {
    try {
      const project = await this.getProject(projectId);

      // Check if already running
      if (this.runningProjects.has(project.id)) {
        throw new Error(`Project is already running: ${project.name}`);
      }

      // Get template for start command
      const template = this.templates.get(project.template);
      const startCommand = template?.commands?.start || 'npm start';

      // Start project process
      const processInfo = await this.processManager.startProject(
        project.id,
        project.path,
        startCommand,
        options
      );

      // Update project status
      await this.updateProject(project.id, { status: 'running' });

      // Track running project
      this.runningProjects.set(project.id, processInfo);

      this.emit('project:started', project);
      this.logger.info(`Project started: ${project.name} (${project.id})`);

      return {
        status: 'running',
        url: processInfo.url,
        pid: processInfo.pid,
        port: processInfo.port,
      };
    } catch (error) {
      this.logger.error('Failed to start project:', error);
      throw error;
    }
  }

  async stopProject(projectId) {
    try {
      const project = await this.getProject(projectId);

      // Check if running
      if (!this.runningProjects.has(project.id)) {
        throw new Error(`Project is not running: ${project.name}`);
      }

      // Stop project process
      await this.processManager.stopProject(project.id);

      // Update project status
      await this.updateProject(project.id, { status: 'stopped' });

      // Remove from running projects
      this.runningProjects.delete(project.id);

      this.emit('project:stopped', project);
      this.logger.info(`Project stopped: ${project.name} (${project.id})`);

      return project;
    } catch (error) {
      this.logger.error('Failed to stop project:', error);
      throw error;
    }
  }

  async getProjectLogs(projectId, options = {}) {
    try {
      const project = await this.getProject(projectId);

      return await this.processManager.getProjectLogs(project.id, options);
    } catch (error) {
      this.logger.error('Failed to get project logs:', error);
      throw error;
    }
  }

  async getRunningProjects() {
    return Array.from(this.runningProjects.keys()).map((projectId) => {
      const project = this.projects.get(projectId);
      const processInfo = this.runningProjects.get(projectId);

      return {
        ...project,
        processInfo,
      };
    });
  }

  async installProjectDependencies(projectId) {
    try {
      const project = await this.getProject(projectId);
      const template = this.templates.get(project.template);
      const installCommand = template?.commands?.install || 'npm install';

      return await this.processManager.runCommand(
        project.path,
        installCommand,
        { stdio: 'inherit' }
      );
    } catch (error) {
      this.logger.error('Failed to install project dependencies:', error);
      throw error;
    }
  }

  async buildProject(projectId) {
    try {
      const project = await this.getProject(projectId);
      const template = this.templates.get(project.template);
      const buildCommand = template?.commands?.build || 'npm run build';

      return await this.processManager.runCommand(project.path, buildCommand, {
        stdio: 'inherit',
      });
    } catch (error) {
      this.logger.error('Failed to build project:', error);
      throw error;
    }
  }

  async testProject(projectId) {
    try {
      const project = await this.getProject(projectId);
      const template = this.templates.get(project.template);
      const testCommand = template?.commands?.test || 'npm test';

      return await this.processManager.runCommand(project.path, testCommand, {
        stdio: 'inherit',
      });
    } catch (error) {
      this.logger.error('Failed to test project:', error);
      throw error;
    }
  }

  // Template management
  async createTemplate(template) {
    try {
      const templatePath = path.join(
        this.config.templatesDir,
        `${template.name}.json`
      );

      await fs.writeFile(templatePath, JSON.stringify(template, null, 2));
      this.templates.set(template.name, template);

      this.logger.info(`Template created: ${template.name}`);
      return template;
    } catch (error) {
      this.logger.error('Failed to create template:', error);
      throw error;
    }
  }

  async listTemplates() {
    return Array.from(this.templates.values());
  }

  async getTemplate(name) {
    const template = this.templates.get(name);
    if (!template) {
      throw new Error(`Template not found: ${name}`);
    }
    return template;
  }

  async deleteTemplate(name) {
    try {
      if (name === 'default') {
        throw new Error('Cannot delete default template');
      }

      const templatePath = path.join(this.config.templatesDir, `${name}.json`);
      await fs.unlink(templatePath);
      this.templates.delete(name);

      this.logger.info(`Template deleted: ${name}`);
    } catch (error) {
      this.logger.error('Failed to delete template:', error);
      throw error;
    }
  }

  async getProjectStatus(projectId) {
    const project = this.projects.get(projectId);
    if (!project) {
      throw new Error('Project not found');
    }

    const isRunning = this.runningProjects.has(projectId);
    const runningInfo = this.runningProjects.get(projectId);

    return {
      id: project.id,
      name: project.name,
      status: isRunning ? 'running' : 'stopped',
      createdAt: project.createdAt,
      lastModified: project.lastModified,
      path: project.path,
      pid: runningInfo?.pid || null,
      uptime: runningInfo ? Date.now() - runningInfo.startTime : 0,
    };
  }

  async stop() {
    try {
      // Stop all running projects
      const runningProjectIds = Array.from(this.runningProjects.keys());

      for (const projectId of runningProjectIds) {
        await this.stopProject(projectId);
      }

      await this.processManager.stop();

      this.logger.info('ProjectManager stopped successfully');
    } catch (error) {
      this.logger.error('Error stopping ProjectManager:', error);
      throw error;
    }
  }
}

export { ProjectManager };
