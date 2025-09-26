// Jira Collector Dashboard App - Industrial Style
class JiraDashboard {
    constructor() {
        this.data = {
            tickets: [],
            projects: [],
            lastUpdate: null
        };
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadData();
        this.startAutoRefresh();
    }

    bindEvents() {
        // Filter controls
        document.getElementById('projectFilter').addEventListener('change', () => {
            this.filterTickets();
        });

        document.getElementById('statusFilter').addEventListener('change', () => {
            this.filterTickets();
        });

        // Time range selector
        document.getElementById('timeRange').addEventListener('change', () => {
            this.updateCharts();
        });
    }

    startAutoRefresh() {
        // Auto-refresh every 5 minutes
        setInterval(() => {
            this.loadData();
        }, 300000);
    }

    async loadData() {
        this.showLoading();
        
        try {
            // Simulate loading delay
            await new Promise(resolve => setTimeout(resolve, 1000));
            
            // Generate sample data
            this.data.tickets = this.generateSampleTickets();
            this.data.projects = [...new Set(this.data.tickets.map(t => t.project))];
            
            this.updateStats();
            this.updateCharts();
            this.updateProjectStats();
            this.updateRecentActivity();
            this.updateTicketList();
            this.updateFilterOptions();
            
            this.data.lastUpdate = new Date();
            this.updateLastUpdateTime();
            
        } catch (error) {
            console.error('Failed to load data:', error);
        } finally {
            this.hideLoading();
        }
    }

    generateSampleTickets() {
        const projects = ['DEV', 'PROJ', 'TEST', 'OPS'];
        const statuses = ['To Do', 'In Progress', 'In Review', 'Done'];
        const priorities = ['High', 'Medium', 'Low'];
        const types = ['Bug', 'Story', 'Task', 'Epic'];
        const assignees = ['ALICE JOHNSON', 'BOB SMITH', 'CAROL DAVIS', 'DAVID WILSON', 'EVA BROWN'];
        
        const tickets = [];
        
        for (let i = 1; i <= 200; i++) {
            const project = projects[Math.floor(Math.random() * projects.length)];
            const status = statuses[Math.floor(Math.random() * statuses.length)];
            const priority = priorities[Math.floor(Math.random() * priorities.length)];
            const type = types[Math.floor(Math.random() * types.length)];
            const assignee = assignees[Math.floor(Math.random() * assignees.length)];
            
            tickets.push({
                key: `${project}-${1000 + i}`,
                project: project,
                summary: `${type}: ${this.generateRandomSummary()}`,
                status: status,
                priority: priority,
                type: type,
                assignee: assignee,
                created: new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000),
                updated: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000)
            });
        }
        
        return tickets;
    }

    generateRandomSummary() {
        const summaries = [
            'AUTHENTICATION MODULE SECURITY UPDATE',
            'DATABASE QUERY OPTIMIZATION REQUIRED',
            'USER INTERFACE REDESIGN IMPLEMENTATION',
            'API ENDPOINT PERFORMANCE MONITORING',
            'UNIT TEST COVERAGE EXPANSION',
            'LEGACY CODE REFACTORING PROJECT',
            'ERROR HANDLING MECHANISM IMPROVEMENT',
            'THIRD PARTY DEPENDENCY UPDATE',
            'CACHING STRATEGY IMPLEMENTATION',
            'LOGGING AND MONITORING ENHANCEMENT',
            'SECURITY VULNERABILITY PATCH',
            'PERFORMANCE BENCHMARKING ANALYSIS',
            'CODE REVIEW PROCESS AUTOMATION',
            'DEPLOYMENT PIPELINE OPTIMIZATION',
            'BACKUP AND RECOVERY PROCEDURES'
        ];
        return summaries[Math.floor(Math.random() * summaries.length)];
    }

    updateStats() {
        const total = this.data.tickets.length;
        const inProgress = this.data.tickets.filter(t => t.status === 'In Progress').length;
        const completed = this.data.tickets.filter(t => t.status === 'Done').length;
        
        document.getElementById('totalTickets').textContent = total.toString().padStart(3, '0');
        document.getElementById('inProgressTickets').textContent = inProgress.toString().padStart(3, '0');
        document.getElementById('completedTickets').textContent = completed.toString().padStart(3, '0');
    }

    updateLastUpdateTime() {
        if (this.data.lastUpdate) {
            const time = this.data.lastUpdate.toLocaleTimeString('en-US', { 
                hour12: false, 
                hour: '2-digit', 
                minute: '2-digit' 
            });
            const date = this.data.lastUpdate.toLocaleDateString('en-US', { 
                month: '2-digit', 
                day: '2-digit' 
            });
            
            document.getElementById('lastUpdateTime').textContent = time;
            document.getElementById('lastUpdateDate').textContent = date;
        }
    }

    updateCharts() {
        this.updateStatusChart();
        this.updatePriorityChart();
    }

    updateStatusChart() {
        const statusCounts = {};
        this.data.tickets.forEach(ticket => {
            statusCounts[ticket.status] = (statusCounts[ticket.status] || 0) + 1;
        });

        const data = [{
            values: Object.values(statusCounts),
            labels: Object.keys(statusCounts),
            type: 'pie',
            hole: 0.4,
            marker: {
                colors: ['#1a1a1a', '#4a4a4a', '#7a7a7a', '#aaaaaa']
            },
            textinfo: 'label+percent',
            textposition: 'outside',
            textfont: {
                size: 12,
                family: 'monospace'
            }
        }];

        const layout = {
            showlegend: false,
            margin: { t: 20, b: 20, l: 20, r: 20 },
            font: { 
                size: 12,
                family: 'sans-serif',
                color: '#2a2a2a'
            },
            paper_bgcolor: 'rgba(0,0,0,0)',
            plot_bgcolor: 'rgba(0,0,0,0)'
        };

        const config = {
            displayModeBar: false,
            responsive: true
        };

        Plotly.newPlot('statusChart', data, layout, config);
    }

    updatePriorityChart() {
        const priorityCounts = {};
        this.data.tickets.forEach(ticket => {
            priorityCounts[ticket.priority] = (priorityCounts[ticket.priority] || 0) + 1;
        });

        const data = [{
            x: Object.keys(priorityCounts),
            y: Object.values(priorityCounts),
            type: 'bar',
            marker: {
                color: ['#1a1a1a', '#4a4a4a', '#7a7a7a']
            },
            text: Object.values(priorityCounts),
            textposition: 'outside',
            textfont: {
                size: 12,
                family: 'monospace',
                color: '#1a1a1a'
            }
        }];

        const layout = {
            xaxis: { 
                title: 'PRIORITY LEVEL',
                titlefont: { size: 12, family: 'monospace' },
                tickfont: { size: 11, family: 'monospace' }
            },
            yaxis: { 
                title: 'TICKET COUNT',
                titlefont: { size: 12, family: 'monospace' },
                tickfont: { size: 11, family: 'monospace' }
            },
            margin: { t: 20, b: 60, l: 80, r: 20 },
            font: { 
                size: 12,
                family: 'sans-serif',
                color: '#2a2a2a'
            },
            paper_bgcolor: 'rgba(0,0,0,0)',
            plot_bgcolor: 'rgba(0,0,0,0)'
        };

        const config = {
            displayModeBar: false,
            responsive: true
        };

        Plotly.newPlot('priorityChart', data, layout, config);
    }

    updateProjectStats() {
        const projectStats = {};
        this.data.tickets.forEach(ticket => {
            if (!projectStats[ticket.project]) {
                projectStats[ticket.project] = 0;
            }
            projectStats[ticket.project]++;
        });

        const html = Object.entries(projectStats).map(([project, count]) => `
            <div class="project-item">
                <div class="project-name">${project}</div>
                <div class="project-count">${count.toString().padStart(3, '0')}</div>
            </div>
        `).join('');

        document.getElementById('projectStats').innerHTML = html;
    }

    updateRecentActivity() {
        const activities = [
            { type: 'collection', text: 'Data collection completed', time: '2 minutes ago' },
            { type: 'update', text: 'Ticket DEV-1245 updated', time: '5 minutes ago' },
            { type: 'system', text: 'System monitoring active', time: '10 minutes ago' },
            { type: 'collection', text: 'Batch processing started', time: '15 minutes ago' },
            { type: 'update', text: 'Ticket PROJ-1342 updated', time: '20 minutes ago' }
        ];

        const html = activities.map(activity => `
            <div class="activity-item">
                <div class="activity-icon">
                    <i class="fas fa-${this.getActivityIcon(activity.type)}" style="font-size: 12px; color: #6a6a6a;"></i>
                </div>
                <div class="activity-content">
                    <div class="activity-text">${activity.text}</div>
                    <div class="activity-time">${activity.time}</div>
                </div>
            </div>
        `).join('');

        document.getElementById('recentActivity').innerHTML = html;
    }

    getActivityIcon(type) {
        const icons = {
            collection: 'sync',
            update: 'edit',
            system: 'cog',
            error: 'exclamation-triangle'
        };
        return icons[type] || 'info-circle';
    }

    updateTicketList() {
        const projectFilter = document.getElementById('projectFilter').value;
        const statusFilter = document.getElementById('statusFilter').value;

        let filteredTickets = this.data.tickets;

        if (projectFilter) {
            filteredTickets = filteredTickets.filter(t => t.project === projectFilter);
        }

        if (statusFilter) {
            filteredTickets = filteredTickets.filter(t => t.status === statusFilter);
        }

        const recentTickets = filteredTickets
            .sort((a, b) => b.updated - a.updated)
            .slice(0, 15);

        const html = recentTickets.map(ticket => `
            <div class="ticket-item">
                <div class="ticket-info">
                    <div class="ticket-key">${ticket.key}</div>
                    <div class="ticket-summary">${ticket.summary}</div>
                    <div class="ticket-meta">
                        <span>${ticket.assignee}</span>
                        <span>${ticket.priority}</span>
                        <span>${ticket.type}</span>
                    </div>
                </div>
                <div class="ticket-status ${this.getStatusClass(ticket.status)}">
                    ${ticket.status.toUpperCase()}
                </div>
            </div>
        `).join('');

        document.getElementById('ticketList').innerHTML = html;
    }

    updateFilterOptions() {
        const projectFilter = document.getElementById('projectFilter');
        const statusFilter = document.getElementById('statusFilter');

        // Update project options
        projectFilter.innerHTML = '<option value="">ALL PROJECTS</option>';
        this.data.projects.forEach(project => {
            const option = document.createElement('option');
            option.value = project;
            option.textContent = project;
            projectFilter.appendChild(option);
        });

        // Update status options
        const statuses = [...new Set(this.data.tickets.map(t => t.status))];
        statusFilter.innerHTML = '<option value="">ALL STATUSES</option>';
        statuses.forEach(status => {
            const option = document.createElement('option');
            option.value = status;
            option.textContent = status.toUpperCase();
            statusFilter.appendChild(option);
        });
    }

    filterTickets() {
        this.updateTicketList();
    }

    getStatusClass(status) {
        const classes = {
            'To Do': 'status-todo',
            'In Progress': 'status-progress',
            'In Review': 'status-review',
            'Done': 'status-done'
        };
        return classes[status] || 'status-todo';
    }

    showLoading() {
        document.getElementById('loadingOverlay').style.display = 'flex';
    }

    hideLoading() {
        document.getElementById('loadingOverlay').style.display = 'none';
    }
}

// Global refresh function
function refreshData() {
    window.dashboard.loadData();
}

// Initialize the dashboard when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.dashboard = new JiraDashboard();
});