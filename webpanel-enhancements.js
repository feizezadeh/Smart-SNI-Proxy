// Web Panel Enhancements
// این فایل قابلیت‌های جدید را به پنل اضافه می‌کند

(function() {
    'use strict';

    // ========== Override displayUsers to show register links ==========
    const originalDisplayUsers = window.displayUsers;

    window.displayUsers = function(users) {
        const container = document.getElementById('usersList');

        if (users.length === 0) {
            container.innerHTML = '<p style="color: #666;">No users registered yet</p>';
            return;
        }

        let html = '<table style="width: 100%; border-collapse: collapse; font-size: 13px;">';
        html += '<tr style="background: #f5f5f5; text-align: left;">';
        html += '<th style="padding: 10px;">Name</th>';
        html += '<th style="min-width: 250px;">Register Link</th>';
        html += '<th>IPs</th><th>Max IPs</th><th>Expires</th><th>Status</th><th>Usage</th><th>Actions</th>';
        html += '</tr>';

        users.forEach(user => {
            const isExpired = new Date(user.expires_at) < new Date();
            const statusColor = user.is_active && !isExpired ? '#27ae60' : '#e74c3c';
            const statusText = user.is_active && !isExpired ? 'Active' : 'Inactive';
            const expiryDate = new Date(user.expires_at).toLocaleDateString();

            // Registration link
            const registerLink = `${window.location.origin}/register?token=${user.id}`;

            // Format IPs list
            const ips = user.ips || [];
            const ipsText = ips.length > 0 ? ips.join('<br>') : '<em style="color: #999;">No IPs yet</em>';
            const ipCount = `${ips.length} / ${user.max_ips}`;

            html += `<tr style="border-bottom: 1px solid #eee;">
                <td style="padding: 10px;">
                    <strong>${user.name}</strong>
                    ${user.description ? '<br><small style="color: #999;">' + user.description + '</small>' : ''}
                </td>
                <td style="font-family: monospace; font-size: 11px; max-width: 300px; word-break: break-all;">
                    <div style="margin-bottom: 5px;">
                        <a href="${registerLink}" target="_blank" style="color: #667eea; text-decoration: none;">${registerLink}</a>
                    </div>
                    <button onclick="copyToClipboard('${registerLink}')" style="padding: 4px 10px; background: #667eea; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 11px;">📋 Copy Link</button>
                </td>
                <td style="font-family: monospace; font-size: 12px;">${ipsText}</td>
                <td style="text-align: center;">${ipCount}</td>
                <td>${expiryDate}</td>
                <td><span style="color: ${statusColor};">● ${statusText}</span></td>
                <td>${user.usage_count || 0}</td>
                <td>
                    <button onclick="extendUser('${user.id}')" style="padding: 5px 10px; margin: 2px; background: #3498db; color: white; border: none; border-radius: 4px; cursor: pointer;">Extend</button>
                    <button onclick="deleteUser('${user.id}')" style="padding: 5px 10px; margin: 2px; background: #e74c3c; color: white; border: none; border-radius: 4px; cursor: pointer;">Delete</button>
                </td>
            </tr>`;
        });

        html += '</table>';
        container.innerHTML = html;
    };

    // ========== Copy to Clipboard Function ==========
    window.copyToClipboard = function(text) {
        navigator.clipboard.writeText(text).then(() => {
            showMessage('✅ Link copied to clipboard!', 'success');
        }).catch(() => {
            // Fallback for older browsers
            const textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.style.position = 'fixed';
            textarea.style.opacity = '0';
            document.body.appendChild(textarea);
            textarea.select();
            try {
                document.execCommand('copy');
                showMessage('✅ Link copied to clipboard!', 'success');
            } catch (err) {
                showMessage('❌ Failed to copy link', 'error');
            }
            document.body.removeChild(textarea);
        });
    };

    // ========== Add Settings Panel ==========
    function addSettingsPanel() {
        // Check if already added
        if (document.querySelector('.settings-panel-container')) {
            console.log('Settings panel already exists');
            return;
        }

        // Find the location to insert settings (before System Info Panel or at end)
        const contentGrid = document.querySelector('.dashboard-container .content-grid');
        if (!contentGrid) {
            console.warn('Content grid not found, trying alternative method');
            // Try finding any container with panels
            const dashboardContainer = document.querySelector('.dashboard-container');
            if (!dashboardContainer) {
                console.error('Dashboard container not found');
                return;
            }
            addSettingsPanelAlternative(dashboardContainer);
            return;
        }

        const settingsHTML = `
            <div class="panel settings-panel-container" style="grid-column: 1 / -1; margin-top: 20px;">
                <h2>⚙️ Settings</h2>
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-top: 20px;">
                    <!-- Change Password -->
                    <div style="padding: 20px; background: #f8f9fa; border-radius: 10px;">
                        <h3 style="margin-bottom: 15px; color: #333;">🔒 Change Password</h3>
                        <div class="form-group">
                            <label>Current Password:</label>
                            <input type="password" id="currentPassword" placeholder="Enter current password" style="width: 100%; padding: 10px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px;">
                        </div>
                        <div class="form-group">
                            <label>New Password:</label>
                            <input type="password" id="newPassword" placeholder="Enter new password" style="width: 100%; padding: 10px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px;">
                        </div>
                        <div class="form-group">
                            <label>Confirm New Password:</label>
                            <input type="password" id="confirmPassword" placeholder="Confirm new password" style="width: 100%; padding: 10px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px;">
                        </div>
                        <button onclick="changePassword()" class="btn btn-primary" style="width: 100%; margin-top: 10px;">Change Password</button>
                    </div>

                    <!-- Change Username -->
                    <div style="padding: 20px; background: #f8f9fa; border-radius: 10px;">
                        <h3 style="margin-bottom: 15px; color: #333;">👤 Change Username</h3>
                        <div class="form-group">
                            <label>Current Password:</label>
                            <input type="password" id="usernamePassword" placeholder="Enter password to confirm" style="width: 100%; padding: 10px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px;">
                        </div>
                        <div class="form-group">
                            <label>New Username:</label>
                            <input type="text" id="newUsername" placeholder="Enter new username" style="width: 100%; padding: 10px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px;">
                        </div>
                        <button onclick="changeUsername()" class="btn btn-primary" style="width: 100%; margin-top: 62px;">Change Username</button>
                        <p style="margin-top: 10px; font-size: 12px; color: #666;">⚠️ You will be logged out after changing username</p>
                    </div>
                </div>
            </div>
        `;

        // Insert before the last child or append
        const systemInfoPanel = Array.from(contentGrid.children).find(el =>
            el.textContent.includes('System Info') || el.querySelector('h2')?.textContent.includes('System Info')
        );

        if (systemInfoPanel) {
            systemInfoPanel.insertAdjacentHTML('beforebegin', settingsHTML);
        } else {
            contentGrid.insertAdjacentHTML('beforeend', settingsHTML);
        }
        console.log('✅ Settings panel added to content grid');
    }

    // Alternative method to add settings panel
    function addSettingsPanelAlternative(container) {
        const settingsHTML = `
            <div class="panel settings-panel-container">
                <h2>⚙️ Settings</h2>
                <div class="settings-grid">
                    <!-- Change Password -->
                    <div class="settings-section">
                        <h3>🔒 Change Password</h3>
                        <div class="form-group">
                            <label for="currentPassword">Current Password</label>
                            <input type="password" id="currentPassword" class="form-control" placeholder="Enter current password">
                        </div>
                        <div class="form-group">
                            <label for="newPassword">New Password</label>
                            <input type="password" id="newPassword" class="form-control" placeholder="Enter new password">
                        </div>
                        <div class="form-group">
                            <label for="confirmPassword">Confirm New Password</label>
                            <input type="password" id="confirmPassword" class="form-control" placeholder="Confirm new password">
                        </div>
                        <button onclick="changePassword()" class="btn btn-primary btn-block">Change Password</button>
                    </div>

                    <!-- Change Username -->
                    <div class="settings-section">
                        <h3>👤 Change Username</h3>
                        <div class="form-group">
                            <label for="usernamePassword">Current Password</label>
                            <input type="password" id="usernamePassword" class="form-control" placeholder="Enter password to confirm">
                        </div>
                        <div class="form-group">
                            <label for="newUsername">New Username</label>
                            <input type="text" id="newUsername" class="form-control" placeholder="Enter new username">
                        </div>
                        <button onclick="changeUsername()" class="btn btn-primary btn-block">Change Username</button>
                        <p class="warning-text">⚠️ You will be logged out after changing username</p>
                    </div>
                </div>
            </div>
        `;

        container.insertAdjacentHTML('beforeend', settingsHTML);
        console.log('✅ Settings panel added via alternative method');
    }

    // ========== Change Password Function ==========
    window.changePassword = async function() {
        const currentPassword = document.getElementById('currentPassword').value;
        const newPassword = document.getElementById('newPassword').value;
        const confirmPassword = document.getElementById('confirmPassword').value;

        if (!currentPassword || !newPassword || !confirmPassword) {
            showMessage('❌ All fields are required', 'error');
            return;
        }

        if (newPassword !== confirmPassword) {
            showMessage('❌ New passwords do not match', 'error');
            return;
        }

        if (newPassword.length < 4) {
            showMessage('❌ Password must be at least 4 characters', 'error');
            return;
        }

        try {
            const sessionId = localStorage.getItem('sessionId') || window.sessionId;
            const response = await fetch('/panel/api/settings/change-password', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Session-ID': sessionId
                },
                body: JSON.stringify({
                    current_password: currentPassword,
                    new_password: newPassword
                })
            });

            if (response.ok) {
                showMessage('✅ Password changed successfully!', 'success');
                document.getElementById('currentPassword').value = '';
                document.getElementById('newPassword').value = '';
                document.getElementById('confirmPassword').value = '';
            } else {
                const data = await response.json();
                showMessage('❌ ' + (data.error || 'Failed to change password'), 'error');
            }
        } catch (error) {
            console.error('Change password error:', error);
            showMessage('❌ Connection error', 'error');
        }
    };

    // ========== Change Username Function ==========
    window.changeUsername = async function() {
        const password = document.getElementById('usernamePassword').value;
        const newUsername = document.getElementById('newUsername').value;

        if (!password || !newUsername) {
            showMessage('❌ All fields are required', 'error');
            return;
        }

        if (newUsername.length < 3) {
            showMessage('❌ Username must be at least 3 characters', 'error');
            return;
        }

        const confirmed = confirm('Are you sure you want to change your username? You will be logged out.');
        if (!confirmed) return;

        try {
            const sessionId = localStorage.getItem('sessionId') || window.sessionId;
            const response = await fetch('/panel/api/settings/change-username', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Session-ID': sessionId
                },
                body: JSON.stringify({
                    password: password,
                    new_username: newUsername
                })
            });

            if (response.ok) {
                showMessage('✅ Username changed successfully! Logging out...', 'success');
                setTimeout(() => {
                    if (typeof logout === 'function') {
                        logout();
                    } else {
                        localStorage.removeItem('sessionId');
                        window.location.reload();
                    }
                }, 2000);
            } else {
                const data = await response.json();
                showMessage('❌ ' + (data.error || 'Failed to change username'), 'error');
            }
        } catch (error) {
            console.error('Change username error:', error);
            showMessage('❌ Connection error', 'error');
        }
    };

    // ========== Initialize on page load ==========
    function initialize() {
        console.log('Initializing web panel enhancements...');

        // Try multiple times with increasing delays
        let attempts = 0;
        const maxAttempts = 20;

        const checkDashboard = setInterval(() => {
            attempts++;
            const dashboard = document.querySelector('.dashboard-container');

            console.log(`Attempt ${attempts}: Dashboard found = ${!!dashboard}, Display = ${dashboard?.style.display}`);

            if (dashboard && (dashboard.style.display !== 'none' || !dashboard.style.display)) {
                clearInterval(checkDashboard);
                console.log('Dashboard detected, adding settings panel...');

                // Try immediately and then with delays
                addSettingsPanel();
                setTimeout(() => addSettingsPanel(), 500);
                setTimeout(() => addSettingsPanel(), 1000);
                setTimeout(() => addSettingsPanel(), 2000);

                console.log('✅ Web panel enhancements loaded');
            } else if (attempts >= maxAttempts) {
                clearInterval(checkDashboard);
                console.warn('Dashboard not found after maximum attempts, trying alternative method');
                // Try adding to body as fallback
                setTimeout(() => {
                    const body = document.body;
                    if (body && !document.querySelector('.settings-panel-container')) {
                        addSettingsPanelAlternative(body);
                    }
                }, 1000);
            }
        }, 200);
    }

    // Start initialization when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initialize);
    } else {
        initialize();
    }

    console.log('Web Panel Enhancements script loaded');
})();
