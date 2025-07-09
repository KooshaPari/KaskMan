/**
 * Project Integration - Integration with Project Management System
 * Handles submission, tracking, and lifecycle management of R&D-generated projects
 */

export class ProjectIntegration {
  constructor(config = {}) {
    this.config = {
      maxPendingProjects: config.maxPendingProjects || 10,
      autoApprovalThreshold: config.autoApprovalThreshold || 0.85,
      notificationEnabled: config.notificationEnabled || true,
      integrationTimeout: config.integrationTimeout || 30000,
      retryAttempts: config.retryAttempts || 3,
      ...config,
    };

    this.projectQueue = new Map();
    this.activeProjects = new Map();
    this.completedProjects = new Map();
    this.integrationAdapters = new Map();

    this.state = {
      totalSubmitted: 0,
      totalApproved: 0,
      totalCompleted: 0,
      averageApprovalTime: 0,
      successRate: 0,
      lastIntegration: null,
    };

    this.initializeAdapters();
  }

  initializeAdapters() {
    // Initialize various project management system adapters
    this.integrationAdapters.set('jira', new JiraAdapter(this.config));
    this.integrationAdapters.set('github', new GitHubAdapter(this.config));
    this.integrationAdapters.set('trello', new TrelloAdapter(this.config));
    this.integrationAdapters.set('asana', new AsanaAdapter(this.config));
    this.integrationAdapters.set('linear', new LinearAdapter(this.config));
    this.integrationAdapters.set('notion', new NotionAdapter(this.config));
    this.integrationAdapters.set(
      'claude-flow',
      new ClaudeFlowAdapter(this.config)
    );

    console.log('🔌 Project Integration adapters initialized');
  }

  async submitSuggestions(suggestions) {
    console.log(`📤 Submitting ${suggestions.length} project suggestions...`);

    const results = {
      submitted: [],
      approved: [],
      rejected: [],
      errors: [],
    };

    for (const suggestion of suggestions) {
      try {
        const result = await this.submitSingleSuggestion(suggestion);
        results.submitted.push(result);

        if (result.status === 'approved') {
          results.approved.push(result);
        } else if (result.status === 'rejected') {
          results.rejected.push(result);
        }
      } catch (error) {
        results.errors.push({
          suggestion: suggestion.id,
          error: error.message,
        });
      }
    }

    // Update statistics
    this.updateStatistics(results);

    // Send notifications if enabled
    if (this.config.notificationEnabled) {
      await this.sendNotifications(results);
    }

    console.log(
      `✅ Project submission complete: ${results.approved.length} approved, ${results.rejected.length} rejected`
    );

    return results;
  }

  async submitSingleSuggestion(suggestion) {
    const projectData = this.formatProjectData(suggestion);

    // Add to queue
    const queueEntry = {
      id: suggestion.id,
      suggestion,
      projectData,
      status: 'pending',
      submittedAt: Date.now(),
      attempts: 0,
    };

    this.projectQueue.set(suggestion.id, queueEntry);

    // Determine target system
    const targetSystem = this.determineTargetSystem(suggestion);

    // Submit to appropriate system
    const result = await this.submitToSystem(projectData, targetSystem);

    // Update queue entry
    queueEntry.status = result.status;
    queueEntry.externalId = result.externalId;
    queueEntry.targetSystem = targetSystem;
    queueEntry.processedAt = Date.now();

    // Move to appropriate collection
    if (result.status === 'approved') {
      this.activeProjects.set(suggestion.id, queueEntry);
      this.projectQueue.delete(suggestion.id);
    } else if (result.status === 'rejected') {
      this.projectQueue.delete(suggestion.id);
    }

    this.state.totalSubmitted++;
    this.state.lastIntegration = Date.now();

    return {
      id: suggestion.id,
      status: result.status,
      externalId: result.externalId,
      targetSystem,
      message: result.message,
      submittedAt: queueEntry.submittedAt,
      processedAt: queueEntry.processedAt,
    };
  }

  formatProjectData(suggestion) {
    return {
      id: suggestion.id,
      title: suggestion.title,
      description: suggestion.description,
      type: suggestion.type,
      category: suggestion.category,
      priority: this.mapPriority(suggestion.priority),
      estimatedDuration: suggestion.estimatedDuration,
      complexity: suggestion.complexity,
      technologies: suggestion.technologies,
      requirements: suggestion.requirements,
      deliverables: suggestion.deliverables,
      successMetrics: suggestion.success_metrics,
      estimatedEffort: suggestion.estimatedEffort,
      requiredSkills: suggestion.requiredSkills,
      dependencies: suggestion.dependencies,
      benefits: suggestion.benefits,
      riskFactors: suggestion.riskFactors,
      feasibilityScore: suggestion.feasibility,
      confidenceScore: suggestion.score,
      sourcePattern: this.formatSourcePattern(suggestion.sourcePattern),
      tags: this.generateTags(suggestion),
      metadata: {
        generatedBy: 'RnD-Module',
        generatedAt: suggestion.timestamp,
        version: '1.0.0',
      },
    };
  }

  mapPriority(priority) {
    const priorityMap = {
      high: 'High',
      medium: 'Medium',
      low: 'Low',
    };
    return priorityMap[priority] || 'Medium';
  }

  formatSourcePattern(pattern) {
    if (!pattern) return null;

    return {
      type: pattern.type,
      confidence: pattern.confidence,
      description: pattern.description || `${pattern.type} pattern`,
      timestamp: pattern.timestamp,
    };
  }

  generateTags(suggestion) {
    const tags = [
      'rnd-generated',
      suggestion.type,
      suggestion.category,
      suggestion.priority,
    ];

    // Add technology tags
    if (suggestion.technologies) {
      suggestion.technologies.forEach((tech) => {
        tags.push(tech.toLowerCase().replace(/\s+/g, '-'));
      });
    }

    // Add complexity tag
    tags.push(`complexity-${suggestion.complexity}`);

    // Add feasibility tag
    if (suggestion.feasibility) {
      const feasibilityLevel =
        suggestion.feasibility > 0.8
          ? 'high'
          : suggestion.feasibility > 0.6
            ? 'medium'
            : 'low';
      tags.push(`feasibility-${feasibilityLevel}`);
    }

    return tags;
  }

  determineTargetSystem(suggestion) {
    // Determine the best target system based on suggestion characteristics

    // High-priority, complex projects go to advanced systems
    if (suggestion.priority === 'high' && suggestion.complexity === 'high') {
      return 'jira';
    }

    // Development-focused projects go to GitHub
    if (
      suggestion.type === 'automation' ||
      suggestion.type === 'optimization'
    ) {
      return 'github';
    }

    // Research projects go to Notion
    if (suggestion.type === 'research') {
      return 'notion';
    }

    // Simple projects go to Trello
    if (suggestion.complexity === 'low') {
      return 'trello';
    }

    // Default to Claude Flow integration
    return 'claude-flow';
  }

  async submitToSystem(projectData, targetSystem) {
    const adapter = this.integrationAdapters.get(targetSystem);
    if (!adapter) {
      throw new Error(`No adapter found for system: ${targetSystem}`);
    }

    try {
      const result = await adapter.submit(projectData);

      // Check for auto-approval
      if (result.status === 'pending' && this.shouldAutoApprove(projectData)) {
        const approvalResult = await adapter.approve(result.externalId);
        return {
          ...result,
          status: 'approved',
          autoApproved: true,
          approvalResult,
        };
      }

      return result;
    } catch (error) {
      console.error(`Failed to submit to ${targetSystem}:`, error);
      throw error;
    }
  }

  shouldAutoApprove(projectData) {
    return (
      projectData.confidenceScore >= this.config.autoApprovalThreshold &&
      projectData.feasibilityScore >= 0.7 &&
      projectData.complexity !== 'high' &&
      !projectData.riskFactors?.length
    );
  }

  async trackProjectProgress() {
    console.log('📊 Tracking project progress...');

    const updates = [];

    for (const [id, project] of this.activeProjects) {
      try {
        const adapter = this.integrationAdapters.get(project.targetSystem);
        if (adapter) {
          const status = await adapter.getStatus(project.externalId);

          if (status.status !== project.status) {
            project.status = status.status;
            project.lastUpdated = Date.now();
            project.progress = status.progress;

            updates.push({
              id,
              oldStatus: project.status,
              newStatus: status.status,
              progress: status.progress,
            });

            // Move to completed if finished
            if (status.status === 'completed') {
              this.completedProjects.set(id, project);
              this.activeProjects.delete(id);
              this.state.totalCompleted++;
            }
          }
        }
      } catch (error) {
        console.error(`Failed to track project ${id}:`, error);
      }
    }

    if (updates.length > 0) {
      console.log(`📈 ${updates.length} project status updates`);
      await this.handleProgressUpdates(updates);
    }

    return updates;
  }

  async handleProgressUpdates(updates) {
    // Process progress updates
    for (const update of updates) {
      const project =
        this.activeProjects.get(update.id) ||
        this.completedProjects.get(update.id);
      if (project) {
        await this.processProjectUpdate(project, update);
      }
    }
  }

  async processProjectUpdate(project, update) {
    // Update internal tracking
    project.progressHistory = project.progressHistory || [];
    project.progressHistory.push({
      timestamp: Date.now(),
      status: update.newStatus,
      progress: update.progress,
    });

    // Send notifications
    if (this.config.notificationEnabled) {
      await this.sendProgressNotification(project, update);
    }

    // Learn from completed projects
    if (update.newStatus === 'completed') {
      await this.learnFromCompletion(project);
    }
  }

  async learnFromCompletion(project) {
    // Extract learning data from completed project
    const learningData = {
      originalSuggestion: project.suggestion,
      actualDuration: this.calculateActualDuration(project),
      successMetrics: await this.evaluateSuccessMetrics(project),
      challenges: await this.identifyChallenges(project),
      improvements: await this.identifyImprovements(project),
    };

    // Store learning data for future improvement
    await this.storeLearningData(learningData);

    console.log(`🧠 Learning data captured for project ${project.id}`);
  }

  calculateActualDuration(project) {
    if (!project.progressHistory || project.progressHistory.length === 0) {
      return null;
    }

    const startTime = project.submittedAt;
    const endTime =
      project.progressHistory[project.progressHistory.length - 1].timestamp;

    return {
      totalDays: Math.ceil((endTime - startTime) / (24 * 60 * 60 * 1000)),
      estimatedDays: this.parseEstimatedDuration(
        project.projectData.estimatedDuration
      ),
      variance: null, // Will be calculated
    };
  }

  parseEstimatedDuration(duration) {
    const match = duration.match(/(\d+)-(\d+)\s*weeks?/);
    if (match) {
      const minWeeks = parseInt(match[1]);
      const maxWeeks = parseInt(match[2]);
      return ((minWeeks + maxWeeks) / 2) * 7; // Average in days
    }
    return null;
  }

  async evaluateSuccessMetrics(project) {
    // Simulate success metrics evaluation
    const metrics = project.projectData.successMetrics || [];

    return metrics.map((metric) => ({
      metric,
      achieved: Math.random() > 0.3, // 70% success rate
      score: Math.random() * 0.5 + 0.5, // 0.5-1.0 range
      notes: `Evaluation of ${metric}`,
    }));
  }

  async identifyChallenges(_project) {
    // Simulate challenge identification
    const challenges = [
      'Technical complexity higher than expected',
      'Resource constraints',
      'Integration difficulties',
      'Timeline pressures',
      'Scope creep',
    ];

    return challenges
      .filter(() => Math.random() > 0.7)
      .map((challenge) => ({
        challenge,
        impact: Math.random() > 0.5 ? 'high' : 'medium',
        resolution: `Addressed through ${challenge.toLowerCase()}`,
      }));
  }

  async identifyImprovements(_project) {
    // Simulate improvement identification
    const improvements = [
      'Better initial complexity estimation',
      'More detailed requirements analysis',
      'Improved technology selection',
      'Enhanced project planning',
      'Better risk assessment',
    ];

    return improvements
      .filter(() => Math.random() > 0.6)
      .map((improvement) => ({
        improvement,
        priority: Math.random() > 0.5 ? 'high' : 'medium',
        impact: 'Better project success rate',
      }));
  }

  async storeLearningData(learningData) {
    // Store in learning database or memory system
    // This would integrate with the main learning system
    console.log('💾 Storing learning data:', learningData);
  }

  updateStatistics(results) {
    this.state.totalSubmitted += results.submitted.length;
    this.state.totalApproved += results.approved.length;

    // Calculate success rate
    if (this.state.totalSubmitted > 0) {
      this.state.successRate =
        this.state.totalApproved / this.state.totalSubmitted;
    }

    // Update average approval time
    const approvalTimes = results.approved
      .filter((r) => r.processedAt && r.submittedAt)
      .map((r) => r.processedAt - r.submittedAt);

    if (approvalTimes.length > 0) {
      const avgTime =
        approvalTimes.reduce((acc, time) => acc + time, 0) /
        approvalTimes.length;
      this.state.averageApprovalTime = avgTime;
    }
  }

  async sendNotifications(results) {
    // Send notifications about project submissions
    const notifications = [];

    if (results.approved.length > 0) {
      notifications.push({
        type: 'success',
        title: 'Projects Approved',
        message: `${results.approved.length} R&D projects have been approved and added to the project queue`,
        projects: results.approved.map((r) => ({ id: r.id, title: r.title })),
      });
    }

    if (results.rejected.length > 0) {
      notifications.push({
        type: 'warning',
        title: 'Projects Rejected',
        message: `${results.rejected.length} R&D projects were rejected`,
        projects: results.rejected.map((r) => ({
          id: r.id,
          title: r.title,
          reason: r.message,
        })),
      });
    }

    if (results.errors.length > 0) {
      notifications.push({
        type: 'error',
        title: 'Submission Errors',
        message: `${results.errors.length} projects failed to submit due to errors`,
        errors: results.errors,
      });
    }

    // Send notifications
    for (const notification of notifications) {
      await this.sendNotification(notification);
    }
  }

  async sendNotification(notification) {
    // Send notification through various channels
    console.log(
      `🔔 Notification: ${notification.title} - ${notification.message}`
    );

    // Could integrate with Slack, email, webhooks, etc.
    // For now, just log
  }

  async sendProgressNotification(project, update) {
    const notification = {
      type: 'info',
      title: 'Project Progress Update',
      message: `Project "${project.suggestion.title}" status changed from ${update.oldStatus} to ${update.newStatus}`,
      project: {
        id: project.id,
        title: project.suggestion.title,
        progress: update.progress,
      },
    };

    await this.sendNotification(notification);
  }

  // Public API methods
  async getProjectStatus(projectId) {
    const project =
      this.activeProjects.get(projectId) ||
      this.completedProjects.get(projectId) ||
      this.projectQueue.get(projectId);

    if (!project) {
      throw new Error(`Project not found: ${projectId}`);
    }

    return {
      id: projectId,
      status: project.status,
      targetSystem: project.targetSystem,
      externalId: project.externalId,
      progress: project.progress,
      submittedAt: project.submittedAt,
      processedAt: project.processedAt,
      lastUpdated: project.lastUpdated,
    };
  }

  async getStatistics() {
    return {
      ...this.state,
      queueSize: this.projectQueue.size,
      activeProjects: this.activeProjects.size,
      completedProjects: this.completedProjects.size,
    };
  }

  async getAllProjects() {
    return {
      pending: Array.from(this.projectQueue.values()),
      active: Array.from(this.activeProjects.values()),
      completed: Array.from(this.completedProjects.values()),
    };
  }

  async retryFailedSubmissions() {
    const failedProjects = Array.from(this.projectQueue.values()).filter(
      (p) => p.status === 'failed' && p.attempts < this.config.retryAttempts
    );

    const results = [];

    for (const project of failedProjects) {
      try {
        project.attempts++;
        const result = await this.submitToSystem(
          project.projectData,
          project.targetSystem
        );
        project.status = result.status;
        project.externalId = result.externalId;
        project.processedAt = Date.now();

        results.push({
          id: project.id,
          status: 'retry_success',
          result,
        });
      } catch (error) {
        results.push({
          id: project.id,
          status: 'retry_failed',
          error: error.message,
        });
      }
    }

    return results;
  }

  async cleanup() {
    // Clean up old entries
    const maxAge = 30 * 24 * 60 * 60 * 1000; // 30 days
    const now = Date.now();

    // Clean up completed projects
    const completedToDelete = [];
    for (const [id, project] of this.completedProjects) {
      if (now - project.processedAt > maxAge) {
        completedToDelete.push(id);
      }
    }

    completedToDelete.forEach((id) => this.completedProjects.delete(id));

    console.log(
      `🧹 Cleaned up ${completedToDelete.length} old completed projects`
    );
  }
}

// Integration adapters for different project management systems
class JiraAdapter {
  constructor(config) {
    this.config = config;
    this.baseUrl = config.jiraUrl || 'https://your-domain.atlassian.net';
    this.auth = config.jiraAuth || null;
  }

  async submit(projectData) {
    // Simulate Jira ticket creation
    const issueData = {
      summary: projectData.title,
      description: this.formatDescription(projectData),
      issuetype: 'Story',
      priority: projectData.priority,
      labels: projectData.tags,
    };

    // Simulate API call
    const response = await this.simulateApiCall(
      'POST',
      '/rest/api/3/issue',
      issueData
    );

    return {
      status: 'pending',
      externalId: response.key,
      message: 'Jira ticket created successfully',
    };
  }

  async approve(_externalId) {
    // Simulate approval process
    return {
      status: 'approved',
      message: 'Project approved in Jira',
    };
  }

  async getStatus(_externalId) {
    // Simulate status check
    const statuses = ['pending', 'in_progress', 'completed'];
    return {
      status: statuses[Math.floor(Math.random() * statuses.length)],
      progress: Math.random() * 100,
    };
  }

  formatDescription(projectData) {
    return `
${projectData.description}

*Generated by R&D Module*

**Requirements:**
${projectData.requirements.map((req) => `- ${req}`).join('\n')}

**Deliverables:**
${projectData.deliverables.map((del) => `- ${del}`).join('\n')}

**Technologies:**
${projectData.technologies.join(', ')}

**Estimated Effort:** ${projectData.estimatedEffort} hours
**Complexity:** ${projectData.complexity}
**Feasibility Score:** ${projectData.feasibilityScore}
    `;
  }

  async simulateApiCall(_method, _endpoint, _data) {
    // Simulate API call with random success/failure
    await new Promise((resolve) => setTimeout(resolve, 100));

    if (Math.random() > 0.1) {
      // 90% success rate
      return {
        key: `RND-${Math.floor(Math.random() * 1000)}`,
        id: Math.floor(Math.random() * 10000),
        self: `${this.baseUrl}${_endpoint}/${Math.floor(Math.random() * 10000)}`,
      };
    } else {
      throw new Error('Jira API call failed');
    }
  }
}

class GitHubAdapter {
  constructor(config) {
    this.config = config;
    this.repo = config.githubRepo || 'owner/repo';
    this.token = config.githubToken || null;
  }

  async submit(projectData) {
    // Simulate GitHub issue creation
    const issueData = {
      title: projectData.title,
      body: this.formatIssueBody(projectData),
      labels: projectData.tags.filter((tag) => tag.length < 50), // GitHub label limit
    };

    const response = await this.simulateApiCall(
      'POST',
      `/repos/${this.repo}/issues`,
      issueData
    );

    return {
      status: 'pending',
      externalId: response.number,
      message: 'GitHub issue created successfully',
    };
  }

  async approve(_externalId) {
    // Add approval label or comment
    return {
      status: 'approved',
      message: 'Project approved via GitHub',
    };
  }

  async getStatus(_externalId) {
    // Check issue status
    const statuses = ['open', 'in_progress', 'closed'];
    return {
      status: statuses[Math.floor(Math.random() * statuses.length)],
      progress: Math.random() * 100,
    };
  }

  formatIssueBody(projectData) {
    return `
${projectData.description}

> 🤖 **Generated by R&D Module**

## Requirements
${projectData.requirements.map((req) => `- [ ] ${req}`).join('\n')}

## Deliverables
${projectData.deliverables.map((del) => `- [ ] ${del}`).join('\n')}

## Technical Details
- **Technologies:** ${projectData.technologies.join(', ')}
- **Estimated Effort:** ${projectData.estimatedEffort} hours
- **Complexity:** ${projectData.complexity}
- **Feasibility Score:** ${projectData.feasibilityScore}

## Success Metrics
${projectData.successMetrics.map((metric) => `- ${metric}`).join('\n')}
    `;
  }

  async simulateApiCall(_method, _endpoint, _data) {
    await new Promise((resolve) => setTimeout(resolve, 150));

    if (Math.random() > 0.05) {
      // 95% success rate
      return {
        number: Math.floor(Math.random() * 1000),
        html_url: `https://github.com/${this.repo}/issues/${Math.floor(Math.random() * 1000)}`,
      };
    } else {
      throw new Error('GitHub API call failed');
    }
  }
}

class TrelloAdapter {
  constructor(config) {
    this.config = config;
    this.boardId = config.trelloBoardId || 'default-board';
    this.token = config.trelloToken || null;
  }

  async submit(projectData) {
    // Simulate Trello card creation
    const cardData = {
      name: projectData.title,
      desc: this.formatCardDescription(projectData),
      idList: this.getTargetList(projectData.priority),
    };

    const response = await this.simulateApiCall('POST', '/1/cards', cardData);

    return {
      status: 'pending',
      externalId: response.id,
      message: 'Trello card created successfully',
    };
  }

  async approve(_externalId) {
    // Move card to approved list
    return {
      status: 'approved',
      message: 'Card moved to approved list',
    };
  }

  async getStatus(_externalId) {
    const statuses = ['pending', 'in_progress', 'completed'];
    return {
      status: statuses[Math.floor(Math.random() * statuses.length)],
      progress: Math.random() * 100,
    };
  }

  formatCardDescription(projectData) {
    return `${projectData.description}\n\n**Tech:** ${projectData.technologies.join(', ')}\n**Effort:** ${projectData.estimatedEffort}h`;
  }

  getTargetList(priority) {
    const lists = {
      High: 'high-priority-list',
      Medium: 'medium-priority-list',
      Low: 'low-priority-list',
    };
    return lists[priority] || 'default-list';
  }

  async simulateApiCall(_method, _endpoint, _data) {
    await new Promise((resolve) => setTimeout(resolve, 100));

    if (Math.random() > 0.05) {
      return {
        id: Math.random().toString(36).substr(2, 9),
        url: `https://trello.com/c/${Math.random().toString(36).substr(2, 9)}`,
      };
    } else {
      throw new Error('Trello API call failed');
    }
  }
}

// Additional adapters (simplified)
class AsanaAdapter {
  constructor(config) {
    this.config = config;
  }
  async submit(_projectData) {
    return {
      status: 'pending',
      externalId: 'asana-' + Date.now(),
      message: 'Asana task created',
    };
  }
  async approve(_externalId) {
    return { status: 'approved', message: 'Asana task approved' };
  }
  async getStatus(_externalId) {
    return { status: 'in_progress', progress: Math.random() * 100 };
  }
}

class LinearAdapter {
  constructor(config) {
    this.config = config;
  }
  async submit(_projectData) {
    return {
      status: 'pending',
      externalId: 'linear-' + Date.now(),
      message: 'Linear issue created',
    };
  }
  async approve(_externalId) {
    return { status: 'approved', message: 'Linear issue approved' };
  }
  async getStatus(_externalId) {
    return { status: 'in_progress', progress: Math.random() * 100 };
  }
}

class NotionAdapter {
  constructor(config) {
    this.config = config;
  }
  async submit(_projectData) {
    return {
      status: 'pending',
      externalId: 'notion-' + Date.now(),
      message: 'Notion page created',
    };
  }
  async approve(_externalId) {
    return { status: 'approved', message: 'Notion page approved' };
  }
  async getStatus(_externalId) {
    return { status: 'in_progress', progress: Math.random() * 100 };
  }
}

class ClaudeFlowAdapter {
  constructor(config) {
    this.config = config;
  }

  async submit(projectData) {
    // Integrate with Claude Flow task system
    const _taskData = {
      title: projectData.title,
      description: projectData.description,
      type: 'rnd_generated',
      priority: projectData.priority.toLowerCase(),
      tags: projectData.tags,
      metadata: projectData.metadata,
    };

    // Simulate Claude Flow task creation
    return {
      status: 'approved', // Claude Flow auto-approves R&D tasks
      externalId: 'cf-' + Date.now(),
      message: 'Claude Flow task created and approved',
    };
  }

  async approve(_externalId) {
    return { status: 'approved', message: 'Claude Flow task approved' };
  }

  async getStatus(_externalId) {
    return { status: 'in_progress', progress: Math.random() * 100 };
  }
}
