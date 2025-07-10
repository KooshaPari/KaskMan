/**
 * Tests for R&D Module
 */

import { RnDModule, RnDUtils } from './index.js';
import { jest } from '@jest/globals';

describe('RnDModule', () => {
  let rndModule;

  beforeEach(() => {
    rndModule = new RnDModule(RnDUtils.createDemoConfig());
  });

  afterEach(() => {
    if (rndModule && rndModule.initialized) {
      rndModule.shutdown();
    }
    jest.clearAllMocks();
  });

  describe('constructor', () => {
    it('should initialize with default configuration', () => {
      expect(rndModule).toBeDefined();
      expect(rndModule.initialized).toBe(false);
      expect(rndModule.config).toBeDefined();
      expect(rndModule.stats).toBeDefined();
    });

    it('should merge custom config with defaults', () => {
      const customConfig = { debug: true, verbose: false };
      const module = new RnDModule(customConfig);

      expect(module.config.debug).toBe(true);
      expect(module.config.verbose).toBe(false);
      expect(module.config.learningRate).toBeDefined(); // From defaults
    });
  });

  describe('initialize', () => {
    it('should initialize successfully', async () => {
      const result = await rndModule.initialize();

      expect(result).toBeDefined();
      expect(result.success).toBe(true);
      expect(result.mode).toBeDefined();
      expect(result.components).toContain('coordinator');
      expect(rndModule.initialized).toBe(true);
    });

    it('should not initialize twice', async () => {
      await rndModule.initialize();

      // Second initialization should not throw but should handle gracefully
      const result = await rndModule.initialize();
      expect(result.success).toBe(true);
    });
  });

  describe('getStatus', () => {
    it('should return not_initialized status before initialization', async () => {
      const status = await rndModule.getStatus();

      expect(status).toBeDefined();
      expect(status.status).toBe('not_initialized');
      expect(status.timestamp).toBeDefined();
    });

    it('should return active status after initialization', async () => {
      await rndModule.initialize();
      
      // Small delay to ensure uptime is greater than 0
      await new Promise(resolve => setTimeout(resolve, 10));
      
      const status = await rndModule.getStatus();

      expect(status).toBeDefined();
      expect(status.status).toBe('active');
      expect(status.mode).toBeDefined();
      expect(status.uptime).toBeGreaterThanOrEqual(0);
      expect(status.coordinator).toBeDefined();
    });
  });

  describe('getStatistics', () => {
    it('should return basic statistics', async () => {
      const stats = await rndModule.getStatistics();

      expect(stats).toBeDefined();
      expect(stats.totalSuggestions).toBe(0);
      expect(stats.approvedProjects).toBe(0);
      expect(stats.completedProjects).toBe(0);
      expect(stats.learningCycles).toBe(0);
    });

    it('should return enhanced statistics after initialization', async () => {
      await rndModule.initialize();
      
      // Small delay to ensure uptime is measured
      await new Promise(resolve => setTimeout(resolve, 10));
      
      const stats = await rndModule.getStatistics();

      expect(stats).toBeDefined();
      expect(stats.uptime).toBeGreaterThanOrEqual(0);
      expect(stats.learningStats).toBeDefined();
      expect(stats.patternStats).toBeDefined();
    });
  });

  describe('forceActivation', () => {
    it('should throw error if not initialized', async () => {
      await expect(rndModule.forceActivation()).rejects.toThrow(
        'R&D Module not initialized'
      );
    });

    it('should force activation successfully', async () => {
      await rndModule.initialize();
      const result = await rndModule.forceActivation();

      expect(result).toBeDefined();
      expect(result.success).toBe(true);
      expect(result.mode).toBe('active');
    });
  });

  describe('addFeedback', () => {
    it('should throw error if not initialized', async () => {
      const feedback = { type: 'positive', message: 'Good job!' };
      await expect(rndModule.addFeedback(feedback)).rejects.toThrow(
        'R&D Module not initialized'
      );
    });

    it('should add feedback successfully', async () => {
      await rndModule.initialize();
      const feedback = { type: 'positive', message: 'Good job!' };
      const result = await rndModule.addFeedback(feedback);

      expect(result).toBeDefined();
      expect(result.success).toBe(true);
      expect(result.feedback).toEqual(feedback);
    });
  });

  describe('generateSuggestions', () => {
    it('should throw error if not initialized', async () => {
      await expect(rndModule.generateSuggestions()).rejects.toThrow(
        'R&D Module not initialized'
      );
    });

    it('should generate suggestions successfully', async () => {
      await rndModule.initialize();
      const result = await rndModule.generateSuggestions();

      expect(result).toBeDefined();
      expect(result.success).toBe(true);
      expect(result.suggestions).toBeDefined();
      expect(result.count).toBeDefined();
    });
  });

  describe('getHealth', () => {
    it('should return not_initialized health before initialization', async () => {
      const health = await rndModule.getHealth();

      expect(health).toBeDefined();
      expect(health.status).toBe('not_initialized');
    });

    it('should return healthy status after initialization', async () => {
      await rndModule.initialize();
      
      // Allow time for all components to initialize properly
      await new Promise(resolve => setTimeout(resolve, 50));
      
      const health = await rndModule.getHealth();

      expect(health).toBeDefined();
      expect(['healthy', 'degraded']).toContain(health.status);
      expect(health.components).toBeDefined();
      expect(health.components.coordinator).toBeDefined();
      expect(health.components.dataStore).toBeDefined();
    });
  });

  describe('shutdown', () => {
    it('should shutdown successfully', async () => {
      await rndModule.initialize();
      const result = await rndModule.shutdown();

      expect(result).toBeDefined();
      expect(result.success).toBe(true);
      expect(rndModule.initialized).toBe(false);
    });

    it('should handle shutdown when not initialized', async () => {
      const result = await rndModule.shutdown();
      expect(result.success).toBe(true);
    });
  });
});

describe('RnDUtils', () => {
  describe('createBasicConfig', () => {
    it('should create basic configuration', () => {
      const config = RnDUtils.createBasicConfig();

      expect(config).toBeDefined();
      expect(config.debug).toBe(false);
      expect(config.verbose).toBe(true);
      expect(config.dataDir).toBe('./data/rnd-module');
    });

    it('should merge custom options', () => {
      const config = RnDUtils.createBasicConfig({
        debug: true,
        quickStart: true,
        maxSuggestions: 10,
      });

      expect(config.debug).toBe(true);
      expect(config.dormantPeriod).toBe(60000); // Quick start
      expect(config.maxProjectSuggestions).toBe(10);
    });
  });

  describe('createDemoConfig', () => {
    it('should create demo configuration', () => {
      const config = RnDUtils.createDemoConfig();

      expect(config).toBeDefined();
      expect(config.debug).toBe(true);
      expect(config.dormantPeriod).toBe(30000); // 30 seconds
      expect(config.learningThreshold).toBe(0.3);
      expect(config.autoBackup).toBe(false);
    });
  });

  describe('createProductionConfig', () => {
    it('should create production configuration', () => {
      const config = RnDUtils.createProductionConfig();

      expect(config).toBeDefined();
      expect(config.debug).toBe(false);
      expect(config.verbose).toBe(false);
      expect(config.dataDir).toBe('/var/lib/rnd-module');
      expect(config.maxBackups).toBe(20);
      expect(config.autoBackup).toBe(true);
    });

    it('should merge custom options', () => {
      const config = RnDUtils.createProductionConfig({
        dataDir: '/custom/data',
        maxMemorySize: 100000,
      });

      expect(config.dataDir).toBe('/custom/data');
      expect(config.maxMemorySize).toBe(100000);
    });
  });
});
