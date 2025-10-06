// Smart SNI Web Panel Enhancements
// Adds registration link features and settings functions

(function() {
    'use strict';

    // ========== Override displayUsers to add Registration Links ==========
    window.displayUsers = function(users) {
        const container = document.getElementById('usersList');

        if (users.length === 0) {
            container.innerHTML = '<p class="text-muted">No users registered yet</p>';
            return;
        }

        let html = '<table class="table table-striped table-hover">';
        html += '<thead><tr>';
        html += '<th>Name</th>';
        html += '<th>Register Link</th>';
        html += '<th>IPs</th>';
        html += '<th>Max IPs</th>';
        html += '<th>Expires</th>';
        html += '<th>Status</th>';
        html += '<th>Usage</th>';
        html += '<th>Actions</th>';
        html += '</tr></thead><tbody>';

        users.forEach(user => {
            const isExpired = new Date(user.expires_at) < new Date();
            const statusColor = user.is_active && !isExpired ? 'success' : 'danger';
            const statusText = user.is_active && !isExpired ? 'Active' : 'Inactive';
            const expiryDate = new Date(user.expires_at).toLocaleDateString();

            const ips = user.ips || [];
            const ipsText = ips.length > 0 ? ips.join('<br>') : '<em class="text-muted">No IPs yet</em>';
            const ipCount = `${ips.length} / ${user.max_ips}`;

            const registerLink = `${window.location.origin}/register?token=${user.id}`;

            html += `<tr>
                <td>
                    <strong>${user.name}</strong>
                    ${user.description ? '<br><small class="text-muted">' + user.description + '</small>' : ''}
                </td>
                <td style="max-width: 300px;">
                    <div class="input-group input-group-sm mb-1">
                        <input type="text" class="form-control form-control-sm" value="${registerLink}" readonly>
                        <button onclick="copyToClipboard('${registerLink}')" class="btn btn-outline-primary btn-sm">
                            <i class="bi bi-clipboard"></i>
                        </button>
                    </div>
                </td>
                <td style="font-family: monospace; font-size: 11px;">${ipsText}</td>
                <td class="text-center">${ipCount}</td>
                <td>${expiryDate}</td>
                <td><span class="badge bg-${statusColor}">${statusText}</span></td>
                <td>${user.usage_count || 0}</td>
                <td>
                    <button onclick="extendUser('${user.id}')" class="btn btn-info btn-sm">Extend</button>
                    <button onclick="deleteUser('${user.id}')" class="btn btn-danger btn-sm">Delete</button>
                </td>
            </tr>`;
        });

        html += '</tbody></table>';
        container.innerHTML = html;
    };

    // ========== Copy to Clipboard ==========
    window.copyToClipboard = function(text) {
        navigator.clipboard.writeText(text).then(() => {
            showToast('✅ Link copied to clipboard!', 'success');
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
            showToast('✅ Link copied to clipboard!', 'success');
        });
    };

    // ========== Change Password Function ==========
    window.changePassword = async function() {
        const currentPassword = document.getElementById('currentPassword').value;
        const newPassword = document.getElementById('newPassword').value;
        const confirmPassword = document.getElementById('confirmPassword').value;

        if (!currentPassword || !newPassword || !confirmPassword) {
            showToast('❌ All fields are required', 'danger');
            return;
        }

        if (newPassword !== confirmPassword) {
            showToast('❌ New passwords do not match', 'danger');
            return;
        }

        if (newPassword.length < 4) {
            showToast('❌ Password must be at least 4 characters', 'danger');
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
                showToast('✅ Password changed successfully!', 'success');
                document.getElementById('currentPassword').value = '';
                document.getElementById('newPassword').value = '';
                document.getElementById('confirmPassword').value = '';
            } else {
                const data = await response.json();
                showToast('❌ ' + (data.error || 'Failed to change password'), 'danger');
            }
        } catch (error) {
            console.error('Change password error:', error);
            showToast('❌ Connection error', 'danger');
        }
    };

    // ========== Change Username Function ==========
    window.changeUsername = async function() {
        const password = document.getElementById('usernamePassword').value;
        const newUsername = document.getElementById('newUsername').value;

        if (!password || !newUsername) {
            showToast('❌ All fields are required', 'danger');
            return;
        }

        if (newUsername.length < 3) {
            showToast('❌ Username must be at least 3 characters', 'danger');
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
                showToast('✅ Username changed successfully! Logging out...', 'success');
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
                showToast('❌ ' + (data.error || 'Failed to change username'), 'danger');
            }
        } catch (error) {
            console.error('Change username error:', error);
            showToast('❌ Connection error', 'danger');
        }
    };

    // ========== Show Toast Notification (Bootstrap-style) ==========
    function showToast(message, type) {
        // Create toast container if it doesn't exist
        let toastContainer = document.getElementById('toastContainer');
        if (!toastContainer) {
            toastContainer = document.createElement('div');
            toastContainer.id = 'toastContainer';
            toastContainer.style.cssText = `
                position: fixed;
                top: 20px;
                right: 20px;
                z-index: 9999;
            `;
            document.body.appendChild(toastContainer);
        }

        // Create toast element
        const toast = document.createElement('div');
        toast.className = `alert alert-${type} alert-dismissible fade show`;
        toast.style.cssText = `
            min-width: 300px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
            margin-bottom: 10px;
        `;
        toast.innerHTML = `
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;

        toastContainer.appendChild(toast);

        // Auto remove after 3 seconds
        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => toast.remove(), 150);
        }, 3000);
    }

    console.log('✅ Web Panel Enhancements loaded');
})();
