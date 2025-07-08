/**
 * Tests for Project Manager
 */

import { ProjectManager } from './project-manager.js';
import { jest } from '@jest/globals';
import { promises as fs } from 'fs';
import path from 'path';

describe('ProjectManager', () => {
  let projectManager;

  beforeEach(() => {
    projectManager = new ProjectManager();
  });

  afterEach(async () => {
    jest.clearAllMocks();

    // Clean up any created project directories
    try {
      const projectsDir = projectManager.config.projectsDir;
      const exists = await fs
        .access(projectsDir)
        .then(() => true)
        .catch(() => false);

      if (exists) {
        const entries = await fs.readdir(projectsDir, { withFileTypes: true });
        for (const entry of entries) {
          if (entry.isDirectory()) {
            const projectPath = path.join(projectsDir, entry.name);
            await fs.rm(projectPath, { recursive: true, force: true });
          }
        }
      }
    } catch (error) {
      // Ignore cleanup errors in tests
      console.warn('Test cleanup warning:', error.message);
    }
  });

  describe('constructor', () => {
    it('should initialize with default configuration', () => {
      expect(projectManager).toBeDefined();
      expect(projectManager.projects).toEqual(new Map());
    });
  });

  describe('createProject', () => {
    it('should create a new project with valid config', async () => {
      const config = {
        name: 'test-project',
        description: 'Test project',
        template: 'default',
      };

      const project = await projectManager.createProject(config);

      expect(project).toBeDefined();
      expect(project.name).toBe(config.name);
      expect(project.description).toBe(config.description);
      expect(project.id).toBeDefined();
      expect(project.createdAt).toBeDefined();
    });

    it('should throw error for invalid project name', async () => {
      const config = {
        name: '',
        description: 'Test project',
      };

      await expect(projectManager.createProject(config)).rejects.toThrow(
        'Project name is required'
      );
    });

    it('should throw error for duplicate project name', async () => {
      const config = {
        name: 'duplicate-project',
        description: 'Test project',
      };

      await projectManager.createProject(config);
      await expect(projectManager.createProject(config)).rejects.toThrow(
        'Project name already exists'
      );
    });
  });

  describe('listProjects', () => {
    it('should return empty array when no projects exist', async () => {
      const projects = await projectManager.listProjects();
      expect(projects).toEqual([]);
    });

    it('should return all projects', async () => {
      const config1 = { name: 'project-1', description: 'Project 1' };
      const config2 = { name: 'project-2', description: 'Project 2' };

      await projectManager.createProject(config1);
      await projectManager.createProject(config2);

      const projects = await projectManager.listProjects();
      expect(projects).toHaveLength(2);
      // Projects are sorted by updatedAt descending, so project-2 comes first
      expect(projects[0].name).toBe('project-2');
      expect(projects[1].name).toBe('project-1');
    });
  });

  describe('startProject', () => {
    it('should start a project successfully', async () => {
      const config = { name: 'test-project', description: 'Test project' };
      const project = await projectManager.createProject(config);

      const result = await projectManager.startProject(project.id);

      expect(result).toBeDefined();
      expect(result.status).toBe('running');
      expect(result.pid).toBeDefined();
    });

    it('should throw error for non-existent project', async () => {
      await expect(
        projectManager.startProject('non-existent-id')
      ).rejects.toThrow('Project not found');
    });
  });

  describe('stopProject', () => {
    it('should stop a running project', async () => {
      const config = { name: 'test-project', description: 'Test project' };
      const project = await projectManager.createProject(config);
      await projectManager.startProject(project.id);

      const result = await projectManager.stopProject(project.id);

      expect(result).toBeDefined();
      expect(result.status).toBe('stopped');
    });

    it('should throw error for non-existent project', async () => {
      await expect(
        projectManager.stopProject('non-existent-id')
      ).rejects.toThrow('Project not found');
    });
  });

  describe('getProjectStatus', () => {
    it('should return project status', async () => {
      const config = { name: 'test-project', description: 'Test project' };
      const project = await projectManager.createProject(config);

      const status = await projectManager.getProjectStatus(project.id);

      expect(status).toBeDefined();
      expect(status.project).toBeDefined();
      expect(status.project.id).toBe(project.id);
      expect(status.status).toBeDefined();
    });

    it('should throw error for non-existent project', async () => {
      await expect(
        projectManager.getProjectStatus('non-existent-id')
      ).rejects.toThrow('Project not found');
    });
  });
});
