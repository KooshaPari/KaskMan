/**
 * R&D Module - Self-Learning Research & Development Engine
 * Main entry point for the R&D Module system
 *
 * Agent 5 Implementation for Swarm-Centralized Auto System
 *
 * Features:
 * - Dormant-to-active learning progression
 * - Advanced pattern recognition and user need analysis
 * - Automatic project proposal generation
 * - Learning from completed projects
 * - Full integration with project management systems
 */

import { RnDCoordinator } from './RnDCoordinator.js';
import { LearningAlgorithm } from './LearningAlgorithm.js';
import { PatternRecognition } from './PatternRecognition.js';
import { ProjectGenerator } from './ProjectGenerator.js';
import { ProjectIntegration } from './ProjectIntegration.js';
import { RnDDataStore } from './RnDDataStore.js';

// Default configuration
const DEFAULT_CONFIG = {
  // Learning algorithm settings
  learningRate: 0.1,
  memoryDecay: 0.95,
  adaptationThreshold: 0.8,
  maxMemorySize: 10000,

  // R&D coordinator settings
  dormantPeriod: 7 * 24 * 60 * 60 * 1000, // 7 days
  learningThreshold: 0.7,
  activationTriggers: 5,
  maxProjectSuggestions: 3,

  // Pattern recognition settings
  minPatternOccurrence: 3,
  patternConfidenceThreshold: 0.6,
  maxPatternAge: 7 * 24 * 60 * 60 * 1000, // 7 days

  // Project generator settings
  maxSuggestions: 5,
  minConfidence: 0.6,
  diversityFactor: 0.7,
  innovationWeight: 0.3,
  practicalityWeight: 0.7,

  // Project integration settings
  maxPendingProjects: 10,
  autoApprovalThreshold: 0.85,
  notificationEnabled: true,
  integrationTimeout: 30000,
  retryAttempts: 3,

  // Data store settings
  dataDir: './data/rnd-module',
  backupDir: './data/rnd-module/backups',
  maxBackups: 10,
  autoBackup: true,
  backupInterval: 24 * 60 * 60 * 1000, // 24 hours

  // System settings
  debug: false,
  verbose: true,
  logLevel: 'info',
};

/**
 * Main R&D Module class that orchestrates all components
 */
export class RnDModule {
  constructor(config = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.coordinator = null;
    this.initialized = false;
    this.startTime = Date.now();

    this.stats = {
      totalSuggestions: 0,
      approvedProjects: 0,
      completedProjects: 0,
      learningCycles: 0,
      patternsSigma: 0,
      uptime: 0,
    };
  }

  /**
   * Initialize the R&D Module system
   */
  async initialize() {
    console.log('🚀 Initializing R&D Module System...');

    try {
      // Initialize coordinator
      this.coordinator = new RnDCoordinator(this.config);
      await this.coordinator.initialize();

      // Set up monitoring
      this.setupMonitoring();

      // Set up cleanup
      this.setupCleanup();

      this.initialized = true;

      console.log('✅ R&D Module System initialized successfully');

      return {
        success: true,
        timestamp: Date.now(),
        mode: this.coordinator.state.mode,
        components: [
          'coordinator',
          'learning',
          'patterns',
          'generator',
          'integration',
          'storage',
        ],
      };
    } catch (error) {
      console.error('❌ Failed to initialize R&D Module:', error);
      throw error;
    }
  }

  /**
   * Get comprehensive system status
   */
  async getStatus() {
    if (!this.initialized) {
      return {
        status: 'not_initialized',
        timestamp: Date.now(),
      };
    }

    try {
      const coordinatorStatus = await this.coordinator.getStatus();
      const stats = await this.getStatistics();

      return {
        status: 'active',
        mode: coordinatorStatus.mode,
        uptime: Date.now() - this.startTime,
        coordinator: coordinatorStatus,
        statistics: stats,
        timestamp: Date.now(),
      };
    } catch (error) {
      return {
        status: 'error',
        error: error.message,
        timestamp: Date.now(),
      };
    }
  }

  /**
   * Get system statistics
   */
  async getStatistics() {
    const stats = { ...this.stats };

    if (this.coordinator) {
      try {
        const coordinatorStats = await this.coordinator.getStatus();
        stats.generatedProjects = coordinatorStats.generatedProjects;
        stats.learningData = coordinatorStats.learningData;
        stats.activationScore = coordinatorStats.activationScore;
        stats.uptime = Date.now() - this.startTime;

        // Get learning algorithm insights
        const insights =
          await this.coordinator.modules.learningAlgorithm.getInsights();
        stats.learningStats = insights.modelStats;
        stats.recommendations = insights.recommendations.length;

        // Get pattern recognition stats
        const patternStats = {
          totalPatterns:
            this.coordinator.modules.patternRecognition.state.totalPatterns,
          activePatterns:
            this.coordinator.modules.patternRecognition.state.activePatterns,
          confidence:
            this.coordinator.modules.patternRecognition.state.confidence,
        };
        stats.patternStats = patternStats;

        // Get project integration stats
        const integrationStats =
          await this.coordinator.modules.projectIntegration.getStatistics();
        stats.integrationStats = integrationStats;

        // Get data store stats
        const dataStoreStats = this.coordinator.modules.dataStore.getStatus();
        stats.dataStoreStats = dataStoreStats;
      } catch (error) {
        console.error('Error getting statistics:', error);
      }
    }

    return stats;
  }

  /**
   * Force activation of the R&D module
   */
  async forceActivation() {
    if (!this.initialized) {
      throw new Error('R&D Module not initialized');
    }

    console.log('⚡ Force activating R&D Module...');

    try {
      await this.coordinator.forceActivation();

      return {
        success: true,
        mode: 'active',
        timestamp: Date.now(),
        message: 'R&D Module force activated',
      };
    } catch (error) {
      console.error('Failed to force activate R&D Module:', error);
      throw error;
    }
  }

  /**
   * Add user feedback to improve learning
   */
  async addFeedback(feedback) {
    if (!this.initialized) {
      throw new Error('R&D Module not initialized');
    }

    try {
      await this.coordinator.addUserFeedback(feedback);

      return {
        success: true,
        timestamp: Date.now(),
        feedback: feedback,
      };
    } catch (error) {
      console.error('Failed to add feedback:', error);
      throw error;
    }
  }

  /**
   * Generate project suggestions manually
   */
  async generateSuggestions() {
    if (!this.initialized) {
      throw new Error('R&D Module not initialized');
    }

    try {
      const suggestions = await this.coordinator.generateProjectSuggestions();
      this.stats.totalSuggestions += suggestions.length;

      return {
        success: true,
        suggestions,
        count: suggestions.length,
        timestamp: Date.now(),
      };
    } catch (error) {
      console.error('Failed to generate suggestions:', error);
      throw error;
    }
  }

  /**
   * Get learning insights
   */
  async getInsights() {
    if (!this.initialized) {
      throw new Error('R&D Module not initialized');
    }

    try {
      const insights =
        await this.coordinator.modules.learningAlgorithm.getInsights();
      const patterns =
        await this.coordinator.modules.patternRecognition.getActivePatterns();

      return {
        learning: insights,
        patterns,
        timestamp: Date.now(),
      };
    } catch (error) {
      console.error('Failed to get insights:', error);
      throw error;
    }
  }

  /**
   * Export all R&D data
   */
  async exportData(format = 'json') {
    if (!this.initialized) {
      throw new Error('R&D Module not initialized');
    }

    try {
      const data = await this.coordinator.modules.dataStore.exportData(format);

      return {
        success: true,
        format,
        data,
        timestamp: Date.now(),
      };
    } catch (error) {
      console.error('Failed to export data:', error);
      throw error;
    }
  }

  /**
   * Import R&D data
   */
  async importData(data, format = 'json') {
    if (!this.initialized) {
      throw new Error('R&D Module not initialized');
    }

    try {
      const result = await this.coordinator.modules.dataStore.importData(
        data,
        format
      );

      return {
        success: true,
        result,
        timestamp: Date.now(),
      };
    } catch (error) {
      console.error('Failed to import data:', error);
      throw error;
    }
  }

  /**
   * Run system maintenance
   */
  async runMaintenance() {
    if (!this.initialized) {
      throw new Error('R&D Module not initialized');
    }

    console.log('🔧 Running R&D Module maintenance...');

    try {
      const results = [];

      // Data store maintenance
      const dataStoreResult =
        await this.coordinator.modules.dataStore.maintenance();
      results.push({ component: 'dataStore', ...dataStoreResult });

      // Pattern recognition cleanup
      this.coordinator.modules.patternRecognition.cleanupOldPatterns();
      results.push({ component: 'patternRecognition', success: true });

      // Learning algorithm optimization
      await this.coordinator.modules.learningAlgorithm.performMemoryCleanup();
      results.push({ component: 'learningAlgorithm', success: true });

      // Project integration cleanup
      await this.coordinator.modules.projectIntegration.cleanup();
      results.push({ component: 'projectIntegration', success: true });

      console.log('✅ R&D Module maintenance completed');

      return {
        success: true,
        results,
        timestamp: Date.now(),
      };
    } catch (error) {
      console.error('Failed to run maintenance:', error);
      throw error;
    }
  }

  /**
   * Get system health status
   */
  async getHealth() {
    if (!this.initialized) {
      return {
        status: 'not_initialized',
        timestamp: Date.now(),
      };
    }

    try {
      const health = {
        status: 'healthy',
        components: {},
        timestamp: Date.now(),
      };

      // Check data store health
      health.components.dataStore =
        await this.coordinator.modules.dataStore.getHealth();

      // Check coordinator health
      health.components.coordinator = {
        status:
          this.coordinator.state.mode === 'active' ? 'healthy' : 'dormant',
        mode: this.coordinator.state.mode,
        activationScore: this.coordinator.state.activationScore,
      };

      // Check learning algorithm health
      health.components.learningAlgorithm = {
        status: 'healthy',
        memorySize:
          this.coordinator.modules.learningAlgorithm.model.memoryBank.size,
        confidence:
          this.coordinator.modules.learningAlgorithm.learningState.confidence,
      };

      // Check pattern recognition health
      health.components.patternRecognition = {
        status: 'healthy',
        totalPatterns:
          this.coordinator.modules.patternRecognition.state.totalPatterns,
        activePatterns:
          this.coordinator.modules.patternRecognition.state.activePatterns,
      };

      // Overall health assessment
      const unhealthyComponents = Object.values(health.components).filter(
        (c) => c.status === 'unhealthy' || c.status === 'error'
      );

      if (unhealthyComponents.length > 0) {
        health.status = 'unhealthy';
      } else {
        const degradedComponents = Object.values(health.components).filter(
          (c) => c.status === 'degraded'
        );
        if (degradedComponents.length > 0) {
          health.status = 'degraded';
        }
      }

      return health;
    } catch (error) {
      return {
        status: 'error',
        error: error.message,
        timestamp: Date.now(),
      };
    }
  }

  /**
   * Setup monitoring for the R&D system
   */
  setupMonitoring() {
    // Monitor statistics
    setInterval(
      () => {
        this.updateStatistics();
      },
      5 * 60 * 1000
    ); // Every 5 minutes

    // Health check
    setInterval(
      async () => {
        const health = await this.getHealth();
        if (health.status === 'unhealthy') {
          console.warn('⚠️  R&D Module health check failed:', health);
        }
      },
      15 * 60 * 1000
    ); // Every 15 minutes
  }

  /**
   * Setup cleanup routines
   */
  setupCleanup() {
    // Periodic maintenance
    setInterval(
      async () => {
        try {
          await this.runMaintenance();
        } catch (error) {
          console.error('Scheduled maintenance failed:', error);
        }
      },
      24 * 60 * 60 * 1000
    ); // Every 24 hours
  }

  /**
   * Update internal statistics
   */
  updateStatistics() {
    this.stats.uptime = Date.now() - this.startTime;
    this.stats.learningCycles++;

    if (this.coordinator) {
      this.stats.patternsSigma =
        this.coordinator.modules.patternRecognition.state.totalPatterns;
    }
  }

  /**
   * Graceful shutdown
   */
  async shutdown() {
    console.log('🔄 Shutting down R&D Module...');

    try {
      if (this.coordinator) {
        await this.coordinator.shutdown();
      }

      this.initialized = false;

      console.log('✅ R&D Module shut down successfully');

      return {
        success: true,
        timestamp: Date.now(),
      };
    } catch (error) {
      console.error('Failed to shutdown R&D Module:', error);
      throw error;
    }
  }
}

// Export individual components for direct use
export {
  RnDCoordinator,
  LearningAlgorithm,
  PatternRecognition,
  ProjectGenerator,
  ProjectIntegration,
  RnDDataStore,
};

// Export default configuration
export { DEFAULT_CONFIG };

// Factory function for easy instantiation
export function createRnDModule(config = {}) {
  return new RnDModule(config);
}

// Utility functions
export const RnDUtils = {
  /**
   * Create a basic configuration for quick setup
   */
  createBasicConfig(options = {}) {
    return {
      ...DEFAULT_CONFIG,
      debug: options.debug || false,
      verbose: options.verbose || true,
      dataDir: options.dataDir || './data/rnd-module',
      dormantPeriod: options.quickStart ? 60000 : DEFAULT_CONFIG.dormantPeriod, // 1 minute for quick start
      learningThreshold:
        options.learningThreshold || DEFAULT_CONFIG.learningThreshold,
      maxProjectSuggestions:
        options.maxSuggestions || DEFAULT_CONFIG.maxProjectSuggestions,
    };
  },

  /**
   * Create a demo configuration for testing
   */
  createDemoConfig() {
    return {
      ...DEFAULT_CONFIG,
      debug: true,
      verbose: true,
      dormantPeriod: 30000, // 30 seconds
      learningThreshold: 0.3, // Lower threshold for demo
      activationTriggers: 2,
      maxProjectSuggestions: 5,
      autoBackup: false, // Disable auto-backup for demo
      notificationEnabled: true,
    };
  },

  /**
   * Create a production configuration
   */
  createProductionConfig(options = {}) {
    return {
      ...DEFAULT_CONFIG,
      debug: false,
      verbose: false,
      dataDir: options.dataDir || '/var/lib/rnd-module',
      backupDir: options.backupDir || '/var/lib/rnd-module/backups',
      maxBackups: 20,
      autoBackup: true,
      backupInterval: 12 * 60 * 60 * 1000, // 12 hours
      learningThreshold: 0.8,
      autoApprovalThreshold: 0.9,
      maxMemorySize: 50000,
      ...options,
    };
  },
};

// Version information
export const VERSION = '1.0.0';
export const BUILD_DATE = new Date().toISOString();

console.log(`🧠 R&D Module v${VERSION} loaded (${BUILD_DATE})`);
