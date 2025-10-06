// Smart SNI Web Panel Enhancements
// Adds Settings panel and registration link features

(function() {
    'use strict';

    // ========== Override displayUsers to add Registration Links ==========
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

            const ips = user.ips || [];
            const ipsText = ips.length > 0 ? ips.join('<br>') : '<em style="color: #999;">No IPs yet</em>';
            const ipCount = `${ips.length} / ${user.max_ips}`;

            const registerLink = `${window.location.origin}/register?token=${user.id}`;

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

    // ========== Copy to Clipboard ==========
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
            document.execCommand('copy');
            document.body.removeChild(textarea);
            showMessage('✅ Link copied to clipboard!', 'success');
        });
    };

    // ========== Add Settings Panel ==========
    function addSettingsPanel() {
        // Check if already added
        if (document.querySelector('.settings-panel-container')) {
            return;
        }

        const dashboard = document.querySelector('.dashboard-container');
        if (!dashboard) {
            return;
        }

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

        // Add to dashboard (append to end)
        dashboard.insertAdjacentHTML('beforeend', settingsHTML);
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

        if (!confirm('⚠️ You will be logged out after changing username. Continue?')) {
            return;
        }

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
                }, 1500);
            } else {
                const data = await response.json();
                showMessage('❌ ' + (data.error || 'Failed to change username'), 'error');
            }
        } catch (error) {
            console.error('Change username error:', error);
            showMessage('❌ Connection error', 'error');
        }
    };

    // ========== Show Message Helper ==========
    function showMessage(message, type) {
        // Try to use existing alert, or create a temporary one
        const alertDiv = document.createElement('div');
        alertDiv.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 15px 25px;
            background: ${type === 'success' ? '#27ae60' : '#e74c3c'};
            color: white;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.2);
            z-index: 10000;
            font-weight: 500;
            animation: slideIn 0.3s ease;
        `;
        alertDiv.textContent = message;
        document.body.appendChild(alertDiv);

        setTimeout(() => {
            alertDiv.style.opacity = '0';
            alertDiv.style.transition = 'opacity 0.3s';
            setTimeout(() => alertDiv.remove(), 300);
        }, 3000);
    }

    // ========== Initialize ==========
    function initialize() {
        // Wait for user to login before adding Settings panel
        const checkInterval = setInterval(() => {
            const dashboard = document.querySelector('.dashboard-container');
            const loginPage = document.getElementById('loginPage');

            // Check if user is logged in (dashboard visible, login hidden)
            const isDashboardVisible = dashboard && dashboard.style.display === 'block';
            const isLoginHidden = loginPage && loginPage.style.display === 'none';

            if (isDashboardVisible && isLoginHidden) {
                clearInterval(checkInterval);

                // Add Settings panel once
                setTimeout(() => {
                    addSettingsPanel();
                    console.log('✅ Settings panel added');
                }, 500);
            }
        }, 300);

        // Stop checking after 15 seconds
        setTimeout(() => clearInterval(checkInterval), 15000);
    }

    // Start when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initialize);
    } else {
        initialize();
    }

    console.log('✅ Web Panel Enhancements loaded');
})();
