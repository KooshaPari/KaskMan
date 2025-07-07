/**
 * R&D Module Demonstration
 * Shows how to use the self-learning R&D module system
 */

import { RnDModule, RnDUtils } from './index.js';

class RnDModuleDemo {
  constructor() {
    this.rndModule = null;
    this.demoStartTime = Date.now();
    this.demoSteps = [];
  }

  async runDemo() {
    console.log('🎯 Starting R&D Module Demonstration...\n');

    try {
      // Step 1: Initialize the R&D Module
      await this.step1_Initialize();

      // Step 2: Show dormant state
      await this.step2_DormantState();

      // Step 3: Force activation
      await this.step3_ForceActivation();

      // Step 4: Generate project suggestions
      await this.step4_GenerateProjects();

      // Step 5: Add user feedback
      await this.step5_AddFeedback();

      // Step 6: Show learning insights
      await this.step6_ShowInsights();

      // Step 7: Export data
      await this.step7_ExportData();

      // Step 8: Show system health
      await this.step8_SystemHealth();

      // Step 9: Run maintenance
      await this.step9_RunMaintenance();

      // Step 10: Final status
      await this.step10_FinalStatus();

      console.log('\n🎉 R&D Module Demonstration completed successfully!');
      console.log(
        `📊 Total demo time: ${this.formatDuration(Date.now() - this.demoStartTime)}`
      );
    } catch (error) {
      console.error('❌ Demo failed:', error);
    } finally {
      // Clean shutdown
      if (this.rndModule) {
        await this.rndModule.shutdown();
      }
    }
  }

  async step1_Initialize() {
    console.log('🚀 Step 1: Initializing R&D Module...');

    // Create demo configuration
    const config = RnDUtils.createDemoConfig();

    // Initialize module
    this.rndModule = new RnDModule(config);
    const result = await this.rndModule.initialize();

    console.log('✅ Initialization result:', result);
    console.log('📋 Configuration:');
    console.log(`   - Dormant period: ${config.dormantPeriod / 1000}s`);
    console.log(`   - Learning threshold: ${config.learningThreshold}`);
    console.log(`   - Max suggestions: ${config.maxProjectSuggestions}`);
    console.log('');

    this.demoSteps.push({
      step: 1,
      name: 'Initialize',
      success: true,
      timestamp: Date.now(),
    });
  }

  async step2_DormantState() {
    console.log('😴 Step 2: Observing Dormant State...');

    const status = await this.rndModule.getStatus();
    console.log('📊 Current status:', {
      mode: status.mode,
      activationScore: status.coordinator.activationScore,
      uptime: this.formatDuration(status.uptime),
    });

    console.log(
      '🔍 The module starts in dormant mode, passively learning from system signals'
    );
    console.log('');

    // Wait a bit to show passive learning
    await this.wait(2000);

    this.demoSteps.push({
      step: 2,
      name: 'Dormant State',
      success: true,
      timestamp: Date.now(),
    });
  }

  async step3_ForceActivation() {
    console.log('⚡ Step 3: Force Activating R&D Module...');

    const result = await this.rndModule.forceActivation();
    console.log('✅ Activation result:', result);

    const status = await this.rndModule.getStatus();
    console.log('📊 Status after activation:', {
      mode: status.mode,
      activationScore: status.coordinator.activationScore,
      generatedProjects: status.coordinator.generatedProjects,
    });

    console.log(
      '🧠 Module is now in active mode, generating project suggestions'
    );
    console.log('');

    this.demoSteps.push({
      step: 3,
      name: 'Force Activation',
      success: true,
      timestamp: Date.now(),
    });
  }

  async step4_GenerateProjects() {
    console.log('🎯 Step 4: Generating Project Suggestions...');

    const result = await this.rndModule.generateSuggestions();
    console.log(`✅ Generated ${result.count} project suggestions:`);

    result.suggestions.forEach((suggestion, index) => {
      console.log(`\n📋 Project ${index + 1}:`);
      console.log(`   Title: ${suggestion.title}`);
      console.log(`   Type: ${suggestion.type}`);
      console.log(`   Category: ${suggestion.category}`);
      console.log(`   Priority: ${suggestion.priority}`);
      console.log(`   Description: ${suggestion.description}`);
      console.log(`   Technologies: ${suggestion.technologies.join(', ')}`);
      console.log(`   Estimated Duration: ${suggestion.estimatedDuration}`);
      console.log(`   Complexity: ${suggestion.complexity}`);
      console.log(
        `   Feasibility Score: ${suggestion.feasibility?.toFixed(2) || 'N/A'}`
      );
      console.log(
        `   Confidence Score: ${suggestion.score?.toFixed(2) || 'N/A'}`
      );

      if (suggestion.benefits) {
        console.log(`   Benefits: ${suggestion.benefits.join(', ')}`);
      }

      if (suggestion.requirements) {
        console.log(`   Requirements: ${suggestion.requirements.length} items`);
      }

      if (suggestion.deliverables) {
        console.log(`   Deliverables: ${suggestion.deliverables.length} items`);
      }
    });

    console.log('');

    this.demoSteps.push({
      step: 4,
      name: 'Generate Projects',
      success: true,
      timestamp: Date.now(),
    });
  }

  async step5_AddFeedback() {
    console.log('👍 Step 5: Adding User Feedback...');

    const feedbackData = {
      rating: 0.8,
      comment: 'Good suggestions, especially the automation projects',
      actionTaken: true,
      timestamp: Date.now(),
    };

    const result = await this.rndModule.addFeedback(feedbackData);
    console.log('✅ Feedback added:', result);

    console.log(
      '🧠 The module will use this feedback to improve future suggestions'
    );
    console.log('');

    this.demoSteps.push({
      step: 5,
      name: 'Add Feedback',
      success: true,
      timestamp: Date.now(),
    });
  }

  async step6_ShowInsights() {
    console.log('🔍 Step 6: Showing Learning Insights...');

    const insights = await this.rndModule.getInsights();

    console.log('📊 Learning Algorithm Insights:');
    console.log(`   - Memory Size: ${insights.learning.modelStats.memorySize}`);
    console.log(
      `   - Experience Buffer: ${insights.learning.modelStats.experienceBuffer}`
    );
    console.log(`   - Clusters: ${insights.learning.modelStats.clusters}`);
    console.log(
      `   - Temporal Patterns: ${insights.learning.modelStats.temporalPatterns}`
    );
    console.log(
      `   - Anomalies Detected: ${insights.learning.modelStats.anomalies}`
    );

    console.log('\n🔍 Pattern Recognition Insights:');
    Object.entries(insights.patterns).forEach(([category, patterns]) => {
      console.log(`   - ${category}: ${patterns.length} patterns`);
    });

    console.log('\n💡 Recommendations:');
    insights.learning.recommendations.forEach((rec, index) => {
      console.log(
        `   ${index + 1}. ${rec.description} (confidence: ${rec.confidence.toFixed(2)})`
      );
    });

    console.log('');

    this.demoSteps.push({
      step: 6,
      name: 'Show Insights',
      success: true,
      timestamp: Date.now(),
    });
  }

  async step7_ExportData() {
    console.log('💾 Step 7: Exporting R&D Data...');

    const result = await this.rndModule.exportData('json');
    console.log(`✅ Data exported successfully (${result.format} format)`);

    const dataSize = JSON.stringify(result.data).length;
    console.log(`📊 Export size: ${this.formatSize(dataSize)}`);

    // Show a sample of the exported data structure
    const sampleData = JSON.parse(result.data);
    console.log('📋 Exported data structure:');
    Object.keys(sampleData).forEach((key) => {
      if (key !== 'exportMetadata') {
        console.log(`   - ${key}: ${typeof sampleData[key]}`);
      }
    });

    console.log('');

    this.demoSteps.push({
      step: 7,
      name: 'Export Data',
      success: true,
      timestamp: Date.now(),
    });
  }

  async step8_SystemHealth() {
    console.log('🏥 Step 8: Checking System Health...');

    const health = await this.rndModule.getHealth();
    console.log(`✅ Overall health: ${health.status}`);

    console.log('📊 Component Health:');
    Object.entries(health.components).forEach(
      ([component, componentHealth]) => {
        console.log(`   - ${component}: ${componentHealth.status}`);
        if (componentHealth.mode) {
          console.log(`     Mode: ${componentHealth.mode}`);
        }
        if (componentHealth.memorySize) {
          console.log(`     Memory Size: ${componentHealth.memorySize}`);
        }
        if (componentHealth.totalPatterns) {
          console.log(`     Total Patterns: ${componentHealth.totalPatterns}`);
        }
      }
    );

    console.log('');

    this.demoSteps.push({
      step: 8,
      name: 'System Health',
      success: true,
      timestamp: Date.now(),
    });
  }

  async step9_RunMaintenance() {
    console.log('🔧 Step 9: Running System Maintenance...');

    const result = await this.rndModule.runMaintenance();
    console.log('✅ Maintenance completed:', result.success);

    console.log('📊 Maintenance Results:');
    result.results.forEach((componentResult) => {
      console.log(
        `   - ${componentResult.component}: ${componentResult.success ? 'Success' : 'Failed'}`
      );
    });

    console.log('');

    this.demoSteps.push({
      step: 9,
      name: 'Run Maintenance',
      success: true,
      timestamp: Date.now(),
    });
  }

  async step10_FinalStatus() {
    console.log('📊 Step 10: Final System Status...');

    const status = await this.rndModule.getStatus();
    const stats = await this.rndModule.getStatistics();

    console.log('🎯 Final Status:');
    console.log(`   - Mode: ${status.mode}`);
    console.log(`   - Uptime: ${this.formatDuration(status.uptime)}`);
    console.log(
      `   - Activation Score: ${status.coordinator.activationScore.toFixed(2)}`
    );
    console.log(
      `   - Generated Projects: ${status.coordinator.generatedProjects}`
    );
    console.log(
      `   - Learning Data Points: ${status.coordinator.learningData}`
    );

    console.log('\n📈 System Statistics:');
    console.log(`   - Total Suggestions: ${stats.totalSuggestions}`);
    console.log(`   - Learning Cycles: ${stats.learningCycles}`);
    console.log(`   - Pattern Count: ${stats.patternsSigma}`);
    console.log(
      `   - Memory Size: ${stats.learningStats?.memorySize || 'N/A'}`
    );
    console.log(`   - Recommendations: ${stats.recommendations || 'N/A'}`);

    if (stats.integrationStats) {
      console.log('\n🔗 Integration Statistics:');
      console.log(`   - Queue Size: ${stats.integrationStats.queueSize}`);
      console.log(
        `   - Active Projects: ${stats.integrationStats.activeProjects}`
      );
      console.log(
        `   - Completed Projects: ${stats.integrationStats.completedProjects}`
      );
      console.log(
        `   - Success Rate: ${(stats.integrationStats.successRate * 100).toFixed(1)}%`
      );
    }

    console.log('');

    this.demoSteps.push({
      step: 10,
      name: 'Final Status',
      success: true,
      timestamp: Date.now(),
    });
  }

  // Utility methods
  async wait(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  formatDuration(ms) {
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);

    if (hours > 0) {
      return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  }

  formatSize(bytes) {
    const sizes = ['B', 'KB', 'MB', 'GB'];
    if (bytes === 0) return '0 B';

    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round((bytes / Math.pow(1024, i)) * 100) / 100 + ' ' + sizes[i];
  }
}

// Self-executing demo
if (import.meta.url === `file://${process.argv[1]}`) {
  const demo = new RnDModuleDemo();
  demo.runDemo().catch(console.error);
}

export { RnDModuleDemo };
