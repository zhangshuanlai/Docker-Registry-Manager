// Docker Registry Manager - Frontend JavaScript

// Global app object
const App = {
    // Configuration
    config: {
        apiBase: '/api',
        refreshInterval: 30000, // 30 seconds
        // ip: '192.168.30.65',
        port: 7000
    },

    // Initialize the application
    init() {
        this.setupEventListeners();
        this.startAutoRefresh();
        console.log('Docker Registry Manager initialized');
    },

    // Setup event listeners
    setupEventListeners() {
        // Handle copy buttons
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('copy-btn') || e.target.closest('.copy-btn')) {
                e.preventDefault();
                const button = e.target.closest('.copy-btn');
                const text = button.getAttribute('data-copy') || button.previousElementSibling.textContent;
                this.copyToClipboard(text);
            }
        });

        // Handle keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + K for search
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                const searchInput = document.getElementById('search-input');
                if (searchInput) {
                    searchInput.focus();
                }
            }

            // Escape to close modals
            if (e.key === 'Escape') {
                this.closeAllModals();
            }
        });
    },

    // Copy text to clipboard
    async copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
            this.showToast('已复制到剪贴板', 'success');
        } catch (err) {
            console.error('Failed to copy text: ', err);
            this.showToast('复制失败，请手动复制', 'error');
        }
    },

    // Show toast notification
    showToast(message, type = 'success') {
        // Remove existing toasts
        const existingToasts = document.querySelectorAll('.toast');
        existingToasts.forEach(toast => toast.remove());

        // Create new toast
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;

        const icon = type === 'success' ? 'fas fa-check-circle' : 'fas fa-exclamation-circle';
        toast.innerHTML = `
            <i class="${icon}"></i>
            <span>${message}</span>
        `;

        document.body.appendChild(toast);

        // Show toast
        setTimeout(() => toast.classList.add('show'), 100);

        // Hide toast after 3 seconds
        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    },

    // Close all modals
    closeAllModals() {
        const modals = document.querySelectorAll('.modal');
        modals.forEach(modal => {
            modal.style.display = 'none';
        });
    },

    // Start auto-refresh for statistics
    startAutoRefresh() {
        if (document.getElementById('repo-count')) {
            // this.refreshStats();
            // setInterval(() => this.refreshStats(), this.config.refreshInterval);
        }
    },

    // Refresh statistics
    async refreshStats() {
        try {
            const response = await fetch(`${this.config.apiBase}/stats`);
            if (!response.ok) throw new Error('Failed to fetch stats');

            const stats = await response.json();

            // Update stats display
            const repoCount = document.getElementById('repo-count');
            const tagCount = document.getElementById('tag-count');
            const totalSize = document.getElementById('total-size');

            if (repoCount) repoCount.textContent = stats.RepositoryCount || 0;
            if (tagCount) tagCount.textContent = stats.TotalTags || 0;
            if (totalSize) totalSize.textContent = stats.TotalSize || 'N/A';

        } catch (error) {
            console.error('Failed to refresh stats:', error);
        }
    },

    // Fetch and display manifest
    async showManifest(repoName, tag) {
        const modal = document.getElementById('manifest-modal');
        const content = document.getElementById('manifest-content');

        if (!modal || !content) return;

        modal.style.display = 'flex';
        content.innerHTML = '<code>加载中...</code>';

        try {
            const response = await fetch(`/v2/${repoName}/manifests/${tag}`, {
                headers: {
                    'Accept': 'application/vnd.docker.distribution.manifest.v2+json'
                }
            });

            if (!response.ok) throw new Error(`HTTP ${response.status}`);

            const data = await response.text();

            try {
                const formatted = JSON.stringify(JSON.parse(data), null, 2);
                content.innerHTML = `<code>${this.escapeHtml(formatted)}</code>`;
            } catch (e) {
                content.innerHTML = `<code>${this.escapeHtml(data)}</code>`;
            }

        } catch (error) {
            content.innerHTML = `<code>加载失败: ${error.message}</code>`;
        }
    },

    // Escape HTML
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    // Format bytes to human readable format
    formatBytes(bytes, decimals = 2) {
        if (bytes === 0) return '0 Bytes';

        const k = 1024;
        const dm = decimals < 0 ? 0 : decimals;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];

        const i = Math.floor(Math.log(bytes) / Math.log(k));

        return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
    },

    // Format date to relative time
    formatRelativeTime(date) {
        const now = new Date();
        const diff = now - date;

        const seconds = Math.floor(diff / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);

        if (days > 0) return `${days} 天前`;
        if (hours > 0) return `${hours} 小时前`;
        if (minutes > 0) return `${minutes} 分钟前`;
        return '刚刚';
    }
};

// Utility functions for global use
window.copyPullCommand = function (repoName, tag = 'latest') {
    const host = window.location.host;
    const command = `docker pull ${host}:${this.config.port}/${repoName}:${tag}`;
    App.copyToClipboard(command);
    App.showToast(`拉取命令已复制: ${tag}`);
};

window.copyToClipboard = function (text) {
    App.copyToClipboard(text);
};

window.showManifest = function (repoName, tag) {
    App.showManifest(repoName, tag);
};

window.closeModal = function () {
    App.closeAllModals();
};

window.showToast = function (message, type = 'success') {
    App.showToast(message, type);
};

// Search functionality
window.setupSearch = function (inputId, itemSelector, nameAttribute = 'data-name') {
    const searchInput = document.getElementById(inputId);
    if (!searchInput) return;

    searchInput.addEventListener('input', function (e) {
        const searchTerm = e.target.value.toLowerCase();
        const items = document.querySelectorAll(itemSelector);

        items.forEach(item => {
            const name = item.getAttribute(nameAttribute) || item.textContent;
            if (name.toLowerCase().includes(searchTerm)) {
                item.style.display = '';
            } else {
                item.style.display = 'none';
            }
        });
    });
};

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    App.init();

    // Setup search if search input exists
    if (document.getElementById('search-input')) {
        setupSearch('search-input', '.repo-item', 'data-name');
    }
});

// Handle page visibility change to pause/resume auto-refresh
document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
        console.log('Page hidden, pausing auto-refresh');
    } else {
        console.log('Page visible, resuming auto-refresh');
        App.refreshStats();
    }
});

// Export for use in other scripts
window.App = App;

