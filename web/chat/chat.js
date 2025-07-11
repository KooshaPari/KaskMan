// KaskMan Project Chat Interface - Claude Desktop Style
class ProjectChatInterface {
    constructor() {
        this.sessionId = null;
        this.currentProject = null;
        this.currentView = 'overview';
        this.isProcessing = false;
        this.chatHistory = [];
        this.tuiComponents = [];
        this.websocket = null;
        this.apiBaseUrl = window.location.origin + '/api/v1';
        this.wsUrl = (window.location.protocol === 'https:' ? 'wss:' : 'ws:') + '//' + window.location.host + '/ws';
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.initializeSession();
        this.loadProjects();
        this.connectWebSocket();
    }

    setupEventListeners() {
        // Message input handling
        const messageInput = document.getElementById('messageInput');
        messageInput.addEventListener('input', this.autoResize.bind(this));
        messageInput.addEventListener('keydown', this.handleKeyDown.bind(this));

        // Send button
        document.getElementById('sendButton').addEventListener('click', this.sendMessage.bind(this));

        // Project selector
        document.getElementById('projectSelect').addEventListener('change', this.switchProject.bind(this));
    }

    async initializeSession() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/sessions`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    user_id: this.generateUUID(), // Generate user ID
                    project_id: null // Start without project
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            this.sessionId = data.session_id;
            
            console.log('Chat session initialized:', this.sessionId);

            // Add welcome message if provided
            if (data.welcome_message) {
                this.displayWelcomeMessage(data.welcome_message);
            }
        } catch (error) {
            console.error('Failed to initialize session:', error);
            // Fall back to local mode
            this.sessionId = this.generateUUID();
            this.loadSampleProject();
        }
    }

    async loadProjects() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/projects`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            this.populateProjectSelector(data.projects);
            
            // Load first project by default if available
            if (data.projects.length > 0) {
                await this.loadProject(data.projects[0].id);
            }
        } catch (error) {
            console.error('Failed to load projects:', error);
            this.loadSampleProject();
        }
    }

    async loadProject(projectId) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/projects/${projectId}`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            this.currentProject = await response.json();
            console.log('Project loaded:', this.currentProject);
        } catch (error) {
            console.error('Failed to load project:', error);
            this.loadSampleProject();
        }
    }

    loadSampleProject() {
        // Fallback sample project data
        this.currentProject = {
            id: 'fintech-app',
            name: 'FinTech Mobile App',
            status: 'executing',
            progress: 0.67,
            health: 'good',
            successProbability: 0.89,
            confidenceLevel: 0.92,
            qualityPrediction: 0.88,
            predictedCompletion: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
            tasks: this.generateSampleTasks(),
            risks: this.generateSampleRisks(),
            resources: this.generateSampleResources()
        };
    }

    connectWebSocket() {
        if (!this.sessionId) {
            console.warn('Cannot connect WebSocket without session ID');
            return;
        }

        try {
            this.websocket = new WebSocket(`${this.wsUrl}?session_id=${this.sessionId}`);
            
            this.websocket.onopen = () => {
                console.log('WebSocket connected');
                this.sendWebSocketMessage('ping', {});
            };

            this.websocket.onmessage = (event) => {
                const message = JSON.parse(event.data);
                this.handleWebSocketMessage(message);
            };

            this.websocket.onclose = () => {
                console.log('WebSocket disconnected');
                // Attempt to reconnect after 5 seconds
                setTimeout(() => this.connectWebSocket(), 5000);
            };

            this.websocket.onerror = (error) => {
                console.error('WebSocket error:', error);
            };
        } catch (error) {
            console.error('Failed to connect WebSocket:', error);
        }
    }

    generateSampleTasks() {
        return [
            { id: 1, name: 'User Authentication System', status: 'completed', priority: 'high' },
            { id: 2, name: 'Payment Integration', status: 'in_progress', priority: 'high' },
            { id: 3, name: 'Dashboard UI', status: 'in_progress', priority: 'medium' },
            { id: 4, name: 'Mobile App Testing', status: 'pending', priority: 'high' },
            { id: 5, name: 'Security Audit', status: 'pending', priority: 'critical' }
        ];
    }

    generateSampleRisks() {
        return [
            { id: 1, type: 'technical', description: 'API rate limiting concerns', level: 'medium' },
            { id: 2, type: 'schedule', description: 'Testing phase may extend timeline', level: 'low' },
            { id: 3, type: 'security', description: 'Payment processing compliance', level: 'high' }
        ];
    }

    generateSampleResources() {
        return [
            { id: 1, name: 'Sarah Chen', role: 'Tech Lead', utilization: 0.85 },
            { id: 2, name: 'Mike Rodriguez', role: 'Frontend Developer', utilization: 0.90 },
            { id: 3, name: 'Anna Kim', role: 'Backend Developer', utilization: 0.75 },
            { id: 4, name: 'David Wilson', role: 'DevOps Engineer', utilization: 0.80 }
        ];
    }

    handleKeyDown(event) {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            this.sendMessage();
        }
    }

    autoResize(element) {
        if (typeof element === 'object' && element.target) {
            element = element.target;
        }
        element.style.height = 'auto';
        element.style.height = Math.min(element.scrollHeight, 120) + 'px';
    }

    async sendMessage() {
        const messageInput = document.getElementById('messageInput');
        const message = messageInput.value.trim();
        
        if (!message || this.isProcessing || !this.sessionId) return;

        this.isProcessing = true;
        messageInput.value = '';
        messageInput.style.height = 'auto';

        // Add user message to chat
        this.addMessage('user', message);

        // Show loading
        this.showLoading(true);

        try {
            // Send message to API
            const response = await fetch(`${this.apiBaseUrl}/chat/message`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    session_id: this.sessionId,
                    message: message,
                    type: 'text'
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            
            // Add assistant response
            this.addMessage('assistant', data.content, data.tui_components, data.quick_actions);

            // Update TUI panel if components provided
            if (data.tui_components && data.tui_components.length > 0) {
                this.updateTUIPanel(data.tui_components);
            }

            // Update suggested questions
            if (data.related_questions && data.related_questions.length > 0) {
                this.updateSuggestedQuestions(data.related_questions);
            }

        } catch (error) {
            console.error('Error sending message:', error);
            // Fall back to local processing
            try {
                const response = await this.processMessage(message);
                this.addMessage('assistant', response.text, response.components, response.actions);
                
                if (response.tuiComponents && response.tuiComponents.length > 0) {
                    this.updateTUIPanel(response.tuiComponents);
                }
            } catch (fallbackError) {
                this.addMessage('assistant', 'Sorry, I encountered an error. Please try again.', null, null, 'error');
            }
        } finally {
            this.showLoading(false);
            this.isProcessing = false;
            messageInput.focus();
        }
    }

    async processMessage(message) {
        // Simulate intelligent message processing
        const intent = this.classifyIntent(message);
        
        switch (intent) {
            case 'project_overview':
                return this.generateProjectOverview();
            case 'project_timeline':
                return this.generateProjectTimeline();
            case 'task_management':
                return this.generateTaskManagement();
            case 'risk_analysis':
                return this.generateRiskAnalysis();
            case 'team_status':
                return this.generateTeamStatus();
            default:
                return this.generateGeneralResponse(message);
        }
    }

    classifyIntent(message) {
        const lowerMessage = message.toLowerCase();
        
        if (lowerMessage.includes('overview') || lowerMessage.includes('status') || lowerMessage.includes('summary')) {
            return 'project_overview';
        } else if (lowerMessage.includes('timeline') || lowerMessage.includes('schedule') || lowerMessage.includes('deadline')) {
            return 'project_timeline';
        } else if (lowerMessage.includes('task') || lowerMessage.includes('todo') || lowerMessage.includes('work')) {
            return 'task_management';
        } else if (lowerMessage.includes('risk') || lowerMessage.includes('problem') || lowerMessage.includes('issue')) {
            return 'risk_analysis';
        } else if (lowerMessage.includes('team') || lowerMessage.includes('resource') || lowerMessage.includes('people')) {
            return 'team_status';
        }
        
        return 'general';
    }

    generateProjectOverview() {
        const project = this.currentProject;
        
        return {
            text: `# 📊 **${project.name}** Project Overview

**Status:** ${project.status} | **Progress:** ${(project.progress * 100).toFixed(1)}% | **Health:** ${project.health}

## Quick Stats
- **Success Probability:** ${(project.successProbability * 100).toFixed(1)}%
- **Predicted Completion:** ${project.predictedCompletion.toLocaleDateString()}
- **Confidence Level:** ${(project.confidenceLevel * 100).toFixed(1)}%
- **Active Tasks:** ${project.tasks.length}
- **Team Members:** ${project.resources.length}

## Current Focus
The project is progressing well with strong team performance. Payment integration is currently the critical path item requiring attention.`,
            components: this.generateOverviewComponents(),
            actions: [
                { id: 'view_timeline', text: '📅 View Timeline', type: 'navigation' },
                { id: 'check_risks', text: '⚠️ Check Risks', type: 'navigation' },
                { id: 'view_tasks', text: '✅ View Tasks', type: 'navigation' },
                { id: 'team_status', text: '👥 Team Status', type: 'navigation' }
            ],
            tuiComponents: [
                this.createProjectMetricsChart(),
                this.createTaskStatusChart(),
                this.createProgressTimeline()
            ]
        };
    }

    generateProjectTimeline() {
        return {
            text: `# 📅 **${this.currentProject.name}** Timeline

## Milestones & Progress
Your project is **${(this.currentProject.progress * 100).toFixed(1)}% complete** with **30 days remaining** until the predicted completion date.

## Critical Path Analysis
The AI has identified **2 critical tasks** that directly impact your delivery date:
- Payment Integration (currently in progress)
- Mobile App Testing (pending start)

## Upcoming Milestones
- **Payment Integration Complete** - Expected in 5 days
- **Beta Testing Phase** - Starting in 10 days
- **Production Deployment** - Planned for ${this.currentProject.predictedCompletion.toLocaleDateString()}`,
            components: this.generateTimelineComponents(),
            actions: [
                { id: 'optimize_timeline', text: '🚀 Optimize Timeline', type: 'action' },
                { id: 'view_dependencies', text: '🔗 View Dependencies', type: 'navigation' },
                { id: 'milestone_details', text: '🎯 Milestone Details', type: 'navigation' }
            ],
            tuiComponents: [
                this.createTimelineChart(),
                this.createMilestoneTable()
            ]
        };
    }

    generateTaskManagement() {
        const tasks = this.currentProject.tasks;
        const completedTasks = tasks.filter(t => t.status === 'completed').length;
        const inProgressTasks = tasks.filter(t => t.status === 'in_progress').length;
        const pendingTasks = tasks.filter(t => t.status === 'pending').length;

        return {
            text: `# ✅ **Task Management** - ${this.currentProject.name}

## Task Overview
- **Total Tasks:** ${tasks.length}
- **Completed:** ${completedTasks} (${(completedTasks / tasks.length * 100).toFixed(1)}%)
- **In Progress:** ${inProgressTasks}
- **Pending:** ${pendingTasks}

## AI Insights
The AI recommends prioritizing Payment Integration completion before starting Mobile App Testing. Current velocity suggests we're on track for the planned timeline.

## High Priority Tasks
${tasks.filter(t => t.priority === 'high' || t.priority === 'critical')
    .map(t => `- **${t.name}** (${t.status})`)
    .join('\n')}`,
            components: this.generateTaskComponents(),
            actions: [
                { id: 'create_task', text: '➕ Create Task', type: 'action' },
                { id: 'optimize_assignments', text: '🎯 Optimize Assignments', type: 'action' },
                { id: 'view_blockers', text: '🚫 View Blockers', type: 'filter' }
            ],
            tuiComponents: [
                this.createTaskTable(),
                this.createTaskPriorityMatrix()
            ]
        };
    }

    generateRiskAnalysis() {
        const risks = this.currentProject.risks;

        return {
            text: `# ⚠️ **Risk Analysis** - ${this.currentProject.name}

## Risk Overview
- **Total Identified Risks:** ${risks.length}
- **High Priority:** ${risks.filter(r => r.level === 'high').length}
- **Medium Priority:** ${risks.filter(r => r.level === 'medium').length}
- **Low Priority:** ${risks.filter(r => r.level === 'low').length}

## Critical Risks Requiring Attention
${risks.filter(r => r.level === 'high')
    .map(r => `- **${r.description}** (${r.type})`)
    .join('\n')}

## AI Recommendations
1. Schedule compliance review for payment processing
2. Begin preliminary testing setup to mitigate schedule risks
3. Monitor API usage patterns to prevent rate limiting issues`,
            components: this.generateRiskComponents(),
            actions: [
                { id: 'create_mitigation', text: '🛡️ Create Mitigation Plan', type: 'action' },
                { id: 'risk_assessment', text: '📊 Full Risk Assessment', type: 'navigation' },
                { id: 'escalate_risk', text: '⬆️ Escalate Critical Risks', type: 'action' }
            ],
            tuiComponents: [
                this.createRiskMatrix(),
                this.createRiskTrendChart()
            ]
        };
    }

    generateTeamStatus() {
        const resources = this.currentProject.resources;
        const avgUtilization = resources.reduce((sum, r) => sum + r.utilization, 0) / resources.length;

        return {
            text: `# 👥 **Team Status** - ${this.currentProject.name}

## Team Overview
- **Team Size:** ${resources.length} members
- **Average Utilization:** ${(avgUtilization * 100).toFixed(1)}%
- **Performance:** Strong across all roles

## Team Members
${resources.map(r => `- **${r.name}** (${r.role}) - ${(r.utilization * 100).toFixed(1)}% utilized`).join('\n')}

## AI Insights
Team is performing well with balanced workload distribution. Consider slight reallocation from DevOps to Frontend to optimize completion timing.`,
            components: this.generateTeamComponents(),
            actions: [
                { id: 'rebalance_workload', text: '⚖️ Rebalance Workload', type: 'action' },
                { id: 'team_performance', text: '📈 Team Performance', type: 'navigation' },
                { id: 'resource_planning', text: '📋 Resource Planning', type: 'navigation' }
            ],
            tuiComponents: [
                this.createTeamUtilizationChart(),
                this.createResourceTable()
            ]
        };
    }

    generateGeneralResponse(message) {
        const responses = [
            "I understand you're asking about the project. Could you be more specific about what aspect you'd like to explore?",
            "That's an interesting question! Let me help you with project insights. What specific information would be most helpful?",
            "I'm here to help with your project management needs. Would you like to see the project overview, timeline, tasks, or risks?",
            "Based on your question, I can provide detailed insights about your project. What would you like to focus on first?"
        ];

        return {
            text: responses[Math.floor(Math.random() * responses.length)],
            components: null,
            actions: [
                { id: 'project_overview', text: '📊 Project Overview', type: 'navigation' },
                { id: 'timeline_view', text: '📅 Timeline', type: 'navigation' },
                { id: 'task_status', text: '✅ Tasks', type: 'navigation' },
                { id: 'risk_check', text: '⚠️ Risks', type: 'navigation' }
            ],
            tuiComponents: []
        };
    }

    // TUI Component Generators
    createProjectMetricsChart() {
        return {
            id: this.generateUUID(),
            type: 'chart',
            title: 'Project Health Metrics',
            chartType: 'radial',
            data: {
                labels: ['Progress', 'Success Probability', 'Confidence', 'Quality'],
                values: [
                    this.currentProject.progress * 100,
                    this.currentProject.successProbability * 100,
                    this.currentProject.confidenceLevel * 100,
                    this.currentProject.qualityPrediction * 100
                ]
            }
        };
    }

    createTaskStatusChart() {
        const tasks = this.currentProject.tasks;
        const statusCounts = {
            completed: tasks.filter(t => t.status === 'completed').length,
            in_progress: tasks.filter(t => t.status === 'in_progress').length,
            pending: tasks.filter(t => t.status === 'pending').length
        };

        return {
            id: this.generateUUID(),
            type: 'chart',
            title: 'Task Status Distribution',
            chartType: 'donut',
            data: {
                labels: ['Completed', 'In Progress', 'Pending'],
                values: [statusCounts.completed, statusCounts.in_progress, statusCounts.pending],
                colors: ['#10b981', '#f59e0b', '#6b7280']
            }
        };
    }

    createTaskTable() {
        return {
            id: this.generateUUID(),
            type: 'table',
            title: 'Task List',
            columns: ['Task', 'Status', 'Priority', 'Action'],
            rows: this.currentProject.tasks.map(task => [
                task.name,
                this.getStatusBadge(task.status),
                this.getPriorityBadge(task.priority),
                '<button class="action-btn">View</button>'
            ])
        };
    }

    createResourceTable() {
        return {
            id: this.generateUUID(),
            type: 'table',
            title: 'Team Resources',
            columns: ['Name', 'Role', 'Utilization', 'Status'],
            rows: this.currentProject.resources.map(resource => [
                resource.name,
                resource.role,
                `${(resource.utilization * 100).toFixed(1)}%`,
                this.getUtilizationStatus(resource.utilization)
            ])
        };
    }

    // Helper methods for rendering
    addMessage(type, content, components = null, actions = null, messageType = 'normal') {
        const messagesContainer = document.getElementById('messagesContainer');
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${type}`;

        const avatar = document.createElement('div');
        avatar.className = 'message-avatar';
        avatar.innerHTML = type === 'user' ? '<i class="fas fa-user"></i>' : '<i class="fas fa-robot"></i>';

        const messageContent = document.createElement('div');
        messageContent.className = 'message-content';

        const messageHeader = document.createElement('div');
        messageHeader.className = 'message-header';
        messageHeader.innerHTML = `
            <span class="sender">${type === 'user' ? 'You' : 'KaskMan AI'}</span>
            <span class="timestamp">${new Date().toLocaleTimeString()}</span>
        `;

        const messageBody = document.createElement('div');
        messageBody.className = 'message-body';
        
        if (content.startsWith('#')) {
            // Render markdown
            messageBody.innerHTML = marked.parse(content);
        } else {
            messageBody.innerHTML = content.replace(/\n/g, '<br>');
        }

        messageContent.appendChild(messageHeader);
        messageContent.appendChild(messageBody);

        // Add quick actions if provided
        if (actions && actions.length > 0) {
            const quickActions = document.createElement('div');
            quickActions.className = 'quick-actions';
            
            actions.forEach(action => {
                const actionBtn = document.createElement('button');
                actionBtn.className = 'action-btn';
                actionBtn.textContent = action.text;
                actionBtn.onclick = () => this.handleQuickAction(action);
                quickActions.appendChild(actionBtn);
            });
            
            messageContent.appendChild(quickActions);
        }

        messageDiv.appendChild(avatar);
        messageDiv.appendChild(messageContent);
        messagesContainer.appendChild(messageDiv);

        // Scroll to bottom
        messagesContainer.scrollTop = messagesContainer.scrollHeight;

        // Store in chat history
        this.chatHistory.push({
            type,
            content,
            timestamp: new Date(),
            components,
            actions
        });
    }

    handleQuickAction(action) {
        if (action.type === 'navigation') {
            this.sendQuickAction(action.id);
        } else if (action.type === 'action') {
            this.executeAction(action.id);
        }
    }

    showLoading(show) {
        const loadingOverlay = document.getElementById('loadingOverlay');
        loadingOverlay.style.display = show ? 'flex' : 'none';
    }

    updateTUIPanel(components) {
        const tuiContent = document.getElementById('tuiContent');
        tuiContent.innerHTML = '';

        components.forEach(component => {
            const componentDiv = this.renderTUIComponent(component);
            tuiContent.appendChild(componentDiv);
        });
    }

    renderTUIComponent(component) {
        const componentDiv = document.createElement('div');
        componentDiv.className = 'tui-component';

        const header = document.createElement('div');
        header.className = 'tui-component-header';
        header.textContent = component.title;

        const content = document.createElement('div');
        content.className = 'tui-component-content';

        if (component.type === 'chart') {
            content.appendChild(this.renderChart(component));
        } else if (component.type === 'table') {
            content.appendChild(this.renderTable(component));
        }

        componentDiv.appendChild(header);
        componentDiv.appendChild(content);

        return componentDiv;
    }

    renderChart(component) {
        const canvas = document.createElement('canvas');
        canvas.width = 350;
        canvas.height = 250;

        // Use Chart.js to render the chart
        setTimeout(() => {
            const ctx = canvas.getContext('2d');
            new Chart(ctx, {
                type: component.chartType === 'radial' ? 'radar' : component.chartType === 'donut' ? 'doughnut' : 'bar',
                data: {
                    labels: component.data.labels,
                    datasets: [{
                        data: component.data.values,
                        backgroundColor: component.data.colors || [
                            '#2563eb', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'
                        ],
                        borderWidth: 2
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            display: true,
                            position: 'bottom'
                        }
                    }
                }
            });
        }, 100);

        return canvas;
    }

    renderTable(component) {
        const table = document.createElement('table');
        table.className = 'data-table';

        const thead = document.createElement('thead');
        const headerRow = document.createElement('tr');
        
        component.columns.forEach(column => {
            const th = document.createElement('th');
            th.textContent = column;
            headerRow.appendChild(th);
        });
        
        thead.appendChild(headerRow);
        table.appendChild(thead);

        const tbody = document.createElement('tbody');
        
        component.rows.forEach(row => {
            const tr = document.createElement('tr');
            
            row.forEach(cell => {
                const td = document.createElement('td');
                td.innerHTML = cell;
                tr.appendChild(td);
            });
            
            tbody.appendChild(tr);
        });
        
        table.appendChild(tbody);

        const tableContainer = document.createElement('div');
        tableContainer.className = 'table-container';
        tableContainer.appendChild(table);

        return tableContainer;
    }

    // Utility methods
    generateUUID() {
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
            const r = Math.random() * 16 | 0;
            const v = c == 'x' ? r : (r & 0x3 | 0x8);
            return v.toString(16);
        });
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    populateProjectSelector(projects) {
        const select = document.getElementById('projectSelect');
        
        // Clear existing options except the first one
        select.innerHTML = '<option value="">Select Project...</option>';
        
        projects.forEach(project => {
            const option = document.createElement('option');
            option.value = project.id;
            option.textContent = project.name;
            select.appendChild(option);
        });
    }

    displayWelcomeMessage(welcomeMessage) {
        // Clear any existing messages and add the welcome message
        document.getElementById('messagesContainer').innerHTML = '';
        this.addMessage('assistant', welcomeMessage.content, null, welcomeMessage.actions);
    }

    handleWebSocketMessage(message) {
        switch (message.type) {
            case 'pong':
                console.log('WebSocket pong received');
                break;
            case 'message_response':
                // Handle real-time message updates
                console.log('Real-time message:', message.data);
                break;
            case 'typing':
                // Handle typing indicators
                this.showTypingIndicator();
                break;
            default:
                console.log('Unknown WebSocket message:', message);
        }
    }

    sendWebSocketMessage(type, data) {
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            this.websocket.send(JSON.stringify({
                type: type,
                session_id: this.sessionId,
                data: data,
                timestamp: new Date()
            }));
        }
    }

    updateSuggestedQuestions(questions) {
        const container = document.getElementById('suggestedQuestions');
        container.innerHTML = '';
        
        questions.forEach(question => {
            const span = document.createElement('span');
            span.className = 'suggestion';
            span.textContent = question;
            span.onclick = () => setSuggestion(question);
            container.appendChild(span);
        });
    }

    showTypingIndicator() {
        // Add typing indicator (implement if needed)
        console.log('User is typing...');
    }

    getStatusBadge(status) {
        const badges = {
            completed: '<span class="badge success">Completed</span>',
            in_progress: '<span class="badge warning">In Progress</span>',
            pending: '<span class="badge secondary">Pending</span>'
        };
        return badges[status] || status;
    }

    getPriorityBadge(priority) {
        const badges = {
            critical: '<span class="badge danger">Critical</span>',
            high: '<span class="badge warning">High</span>',
            medium: '<span class="badge info">Medium</span>',
            low: '<span class="badge secondary">Low</span>'
        };
        return badges[priority] || priority;
    }

    getUtilizationStatus(utilization) {
        if (utilization > 0.9) return '<span class="badge danger">Overloaded</span>';
        if (utilization > 0.8) return '<span class="badge warning">Busy</span>';
        if (utilization > 0.6) return '<span class="badge success">Optimal</span>';
        return '<span class="badge secondary">Available</span>';
    }

    // Additional component generators
    generateOverviewComponents() { return []; }
    generateTimelineComponents() { return []; }
    generateTaskComponents() { return []; }
    generateRiskComponents() { return []; }
    generateTeamComponents() { return []; }
    createProgressTimeline() { return this.createTaskStatusChart(); }
    createTimelineChart() { return this.createProjectMetricsChart(); }
    createMilestoneTable() { return this.createTaskTable(); }
    createTaskPriorityMatrix() { return this.createTaskStatusChart(); }
    createRiskMatrix() { return this.createTaskStatusChart(); }
    createRiskTrendChart() { return this.createProjectMetricsChart(); }
    createTeamUtilizationChart() { return this.createProjectMetricsChart(); }
}

// Global functions for HTML event handlers
let chatInterface;

function sendQuickAction(action) {
    const actionMessages = {
        project_overview: 'Show me the project overview',
        timeline_view: 'Show me the project timeline',
        task_status: 'What\'s the status of our tasks?',
        risk_check: 'What are the current risks?',
        team_status: 'How is the team performing?'
    };
    
    const messageInput = document.getElementById('messageInput');
    messageInput.value = actionMessages[action] || action;
    sendMessage();
}

function sendMessage() {
    if (chatInterface) {
        chatInterface.sendMessage();
    }
}

function handleKeyDown(event) {
    if (chatInterface) {
        chatInterface.handleKeyDown(event);
    }
}

function autoResize(element) {
    if (chatInterface) {
        chatInterface.autoResize(element);
    }
}

function setSuggestion(suggestion) {
    document.getElementById('messageInput').value = suggestion;
    document.getElementById('messageInput').focus();
}

async function switchProject() {
    const select = document.getElementById('projectSelect');
    if (chatInterface && select.value) {
        try {
            await chatInterface.loadProject(select.value);
            chatInterface.addMessage('assistant', `Switched to project: ${select.options[select.selectedIndex].text}`);
        } catch (error) {
            console.error('Failed to switch project:', error);
            chatInterface.addMessage('assistant', 'Failed to switch project. Please try again.');
        }
    }
}

function switchView(view) {
    // Update navigation
    document.querySelectorAll('.nav-item').forEach(item => item.classList.remove('active'));
    event.target.classList.add('active');
    
    if (chatInterface) {
        chatInterface.currentView = view;
        document.getElementById('chatTitle').textContent = `💬 Project Chat - ${view.charAt(0).toUpperCase() + view.slice(1)} Mode`;
    }
}

function toggleTUIMode() {
    const tuiPanel = document.getElementById('tuiPanel');
    const isVisible = tuiPanel.style.display !== 'none';
    tuiPanel.style.display = isVisible ? 'none' : 'flex';
}

function clearChat() {
    if (confirm('Are you sure you want to clear the chat history?')) {
        document.getElementById('messagesContainer').innerHTML = '';
        if (chatInterface) {
            chatInterface.chatHistory = [];
        }
    }
}

function exportChat() {
    if (chatInterface && chatInterface.chatHistory.length > 0) {
        const chatData = JSON.stringify(chatInterface.chatHistory, null, 2);
        const blob = new Blob([chatData], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `chat-export-${new Date().toISOString().split('T')[0]}.json`;
        a.click();
        URL.revokeObjectURL(url);
    }
}

// Initialize chat interface when page loads
document.addEventListener('DOMContentLoaded', function() {
    chatInterface = new ProjectChatInterface();
    
    // Add badge styles to CSS
    const style = document.createElement('style');
    style.textContent = `
        .badge {
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 500;
            display: inline-block;
        }
        .badge.success { background: #dcfce7; color: #166534; }
        .badge.warning { background: #fef3c7; color: #92400e; }
        .badge.danger { background: #fecaca; color: #991b1b; }
        .badge.info { background: #dbeafe; color: #1e40af; }
        .badge.secondary { background: #f1f5f9; color: #475569; }
    `;
    document.head.appendChild(style);
});