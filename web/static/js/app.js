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
        this.initDescriptionEditor();
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
            // Handle logout button
            if (e.target.id === 'logout-btn') {
                e.preventDefault();
                this.handleLogout();
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
    },

    // Handle logout
    async handleLogout() {
        try {
            const resp = await fetch('/api/logout', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            if (resp.ok) {
                window.location.href = '/'; // 登出后跳转到主页
            } else {
                this.showToast('登出失败', 'error');
            }
        } catch (err) {
            this.showToast('网络错误，登出失败', 'error');
        }
    },

    // Initialize repository description editor
    initDescriptionEditor() {
        const repoNameMatch = window.location.pathname.match(/\/repositories\/([^/]+)/);
        if (!repoNameMatch) return; // Not on a repository page

        const repoName = repoNameMatch[1];

        const renderedDescriptionDiv = document.getElementById('rendered-description');
        const descriptionDisplay = document.getElementById('description-display');
        const descriptionEditorContainer = document.getElementById('description-editor-container');
        const descriptionEditor = document.getElementById('description-editor');
        const editButton = document.getElementById('edit-description-btn');
        const saveButton = document.getElementById('save-description-btn');
        const cancelButton = document.getElementById('cancel-description-btn');

        let currentDescription = ""; // Store the current raw Markdown

        // Function to render and display description
        const renderAndDisplay = (markdown) => {
            if (renderedDescriptionDiv) {
                renderedDescriptionDiv.innerHTML = marked.parse(markdown);
                descriptionDisplay.style.display = 'block';
                descriptionEditorContainer.style.display = 'none';
                currentDescription = markdown; // Update current raw description
                // Update the data-raw-description attribute if needed (e.g., after save)
                if (renderedDescriptionDiv.dataset.rawDescription !== undefined) {
                    renderedDescriptionDiv.dataset.rawDescription = markdown;
                }
            }
        };

        // Initial render if description exists
        if (renderedDescriptionDiv && renderedDescriptionDiv.dataset.rawDescription !== undefined) {
            const initialDescription = renderedDescriptionDiv.dataset.rawDescription;
            renderAndDisplay(initialDescription);
        }

        // Event listener for Edit button
        if (editButton) {
            editButton.addEventListener('click', () => {
                descriptionEditor.value = currentDescription;
                descriptionDisplay.style.display = 'none';
                descriptionEditorContainer.style.display = 'block';
            });
        }

        // Event listener for Save button
        if (saveButton) {
            saveButton.addEventListener('click', async () => {
                const newDescription = descriptionEditor.value;
                try {
                    const response = await fetch(`${App.config.apiBase}/repositories/${repoName}/description`, {
                        method: 'PUT',
                        headers: { 'Content-Type': 'text/plain' }, // Send as plain text
                        body: newDescription
                    });

                    if (response.ok) {
                        App.showToast('仓库说明已保存', 'success');
                        renderAndDisplay(newDescription);
                    } else {
                        const errorText = await response.text();
                        App.showToast(`保存失败: ${errorText || response.statusText}`, 'error');
                    }
                } catch (error) {
                    App.showToast(`网络错误，保存失败: ${error.message}`, 'error');
                }
            });
        }

        // Event listener for Cancel button
        if (cancelButton) {
            cancelButton.addEventListener('click', () => {
                renderAndDisplay(currentDescription); // Revert to original and display
            });
        }
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

    // 登录表单提交逻辑
    if (document.getElementById('login-form')) {
        document.getElementById('login-form').addEventListener('submit', async function (e) {
            e.preventDefault();
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const errorDiv = document.getElementById('login-error');
            errorDiv.textContent = '';
            try {
                const resp = await fetch('/api/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username, password })
                });
                if (resp.ok) {
                    window.location.href = '/';
                } else {
                    const msg = resp.status === 401 ? '用户名或密码错误' : '登录失败';
                    errorDiv.textContent = msg;
                }
            } catch (err) {
                errorDiv.textContent = '网络错误，请重试';
            }
        });
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

