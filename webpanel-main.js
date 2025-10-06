// SmartSNI Web Panel Main JavaScript
// Session Management
let sessionId = localStorage.getItem('sessionId');
let refreshInterval;

// Check if already logged in
if (sessionId) {
    validateSession();
} else {
    showLogin();
}

// ========== Authentication Functions ==========

function showLogin() {
    document.getElementById('loginPage').style.display = 'flex';
    document.getElementById('dashboard').style.display = 'none';
}

function showDashboard(username) {
    document.getElementById('loginPage').style.display = 'none';
    document.getElementById('dashboard').style.display = 'block';
    document.getElementById('currentUser').textContent = username;
    loadDashboardData();
    startAutoRefresh();
}

document.addEventListener('DOMContentLoaded', function() {
    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;

            try {
                const response = await fetch('/panel/api/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username, password })
                });

                const data = await response.json();

                if (response.ok) {
                    sessionId = data.session_id;
                    localStorage.setItem('sessionId', sessionId);
                    showDashboard(username);
                } else {
                    showError('loginError', data.error || 'Login failed');
                }
            } catch (error) {
                showError('loginError', 'Connection error');
            }
        });
    }
});

async function validateSession() {
    try {
        const response = await fetch('/panel/api/validate', {
            headers: { 'X-Session-ID': sessionId }
        });

        if (response.ok) {
            const data = await response.json();
            showDashboard(data.username);
        } else {
            localStorage.removeItem('sessionId');
            showLogin();
        }
    } catch (error) {
        localStorage.removeItem('sessionId');
        showLogin();
    }
}

async function logout() {
    try {
        await fetch('/panel/api/logout', {
            method: 'POST',
            headers: { 'X-Session-ID': sessionId }
        });
    } catch (error) {
        console.error('Logout error:', error);
    }

    localStorage.removeItem('sessionId');
    stopAutoRefresh();
    showLogin();
}

// ========== Dashboard Data Loading ==========

async function loadDashboardData() {
    await Promise.all([
        loadMetrics(),
        loadDomains(),
        loadHealth(),
        loadUsers()
    ]);
}

async function loadMetrics() {
    try {
        const response = await fetch('/panel/api/metrics', {
            headers: { 'X-Session-ID': sessionId }
        });

        if (response.ok) {
            const data = await response.json();
            document.getElementById('dohQueries').textContent = data.doh_queries.toLocaleString();
            document.getElementById('dotQueries').textContent = data.dot_queries.toLocaleString();
            document.getElementById('sniConnections').textContent = data.sni_connections.toLocaleString();

            const total = data.cache_hits + data.cache_misses;
            const hitRate = total > 0 ? ((data.cache_hits / total) * 100).toFixed(1) : 0;
            document.getElementById('cacheHitRate').textContent = hitRate + '%';
        }
    } catch (error) {
        console.error('Error loading metrics:', error);
    }
}

async function loadDomains() {
    try {
        const response = await fetch('/panel/api/domains', {
            headers: { 'X-Session-ID': sessionId }
        });

        if (response.ok) {
            const data = await response.json();
            const container = document.getElementById('domainsList');

            if (Object.keys(data.domains).length === 0) {
                container.innerHTML = '<p style="color: #999; text-align: center; padding: 20px;">No domains configured</p>';
                return;
            }

            container.innerHTML = Object.entries(data.domains).map(([domain, ip]) => `
                <div class="domain-item">
                    <span class="domain-name">${domain}</span>
                    <span class="domain-ip">${ip}</span>
                    <button onclick="removeDomain('${domain}')" class="btn-small btn-danger">Remove</button>
                </div>
            `).join('');
        }
    } catch (error) {
        console.error('Error loading domains:', error);
    }
}

async function loadHealth() {
    try {
        const response = await fetch('/panel/api/health', {
            headers: { 'X-Session-ID': sessionId }
        });

        if (response.ok) {
            const data = await response.json();
            document.getElementById('systemVersion').textContent = data.version;
            document.getElementById('systemUptime').textContent = formatUptime(data.uptime);
            document.getElementById('systemStatus').style.color = data.status === 'healthy' ? '#27ae60' : '#e74c3c';
            document.getElementById('systemStatusText').textContent = data.status.charAt(0).toUpperCase() + data.status.slice(1);
        }
    } catch (error) {
        console.error('Error loading health:', error);
    }
}

// ========== Domain Management Functions ==========

async function addDomain() {
    const domain = document.getElementById('newDomain').value.trim();

    if (!domain) {
        alert('Please enter a domain');
        return;
    }

    try {
        const response = await fetch('/panel/api/domains/add', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Session-ID': sessionId
            },
            body: JSON.stringify({ domain })
        });

        if (response.ok) {
            document.getElementById('newDomain').value = '';
            await loadDomains();
        } else {
            const data = await response.json();
            alert(data.error || 'Failed to add domain');
        }
    } catch (error) {
        alert('Connection error');
    }
}

async function removeDomain(domain) {
    if (!confirm(`Remove domain ${domain}?`)) return;

    try {
        const response = await fetch('/panel/api/domains/remove', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Session-ID': sessionId
            },
            body: JSON.stringify({ domain })
        });

        if (response.ok) {
            await loadDomains();
        } else {
            const data = await response.json();
            alert(data.error || 'Failed to remove domain');
        }
    } catch (error) {
        alert('Connection error');
    }
}

async function reloadConfig() {
    if (!confirm('Reload configuration? This will apply changes without restarting.')) return;

    try {
        const response = await fetch('/panel/api/reload', {
            method: 'POST',
            headers: { 'X-Session-ID': sessionId }
        });

        if (response.ok) {
            alert('Configuration reloaded successfully!');
            await loadDashboardData();
        } else {
            const data = await response.json();
            alert(data.error || 'Failed to reload configuration');
        }
    } catch (error) {
        alert('Connection error');
    }
}

// ========== Auto Refresh ==========

function startAutoRefresh() {
    refreshInterval = setInterval(() => {
        loadMetrics();
        loadHealth();
    }, 5000); // Refresh every 5 seconds
}

function stopAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
}

// ========== Utility Functions ==========

function formatUptime(seconds) {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);

    if (days > 0) return `${days}d ${hours}h ${mins}m`;
    if (hours > 0) return `${hours}h ${mins}m`;
    return `${mins}m`;
}

function showError(elementId, message) {
    const el = document.getElementById(elementId);
    el.textContent = message;
    el.style.display = 'block';
    setTimeout(() => {
        el.style.display = 'none';
    }, 5000);
}

// ========== User Management Functions ==========

async function loadUsers() {
    try {
        const response = await fetch('/panel/api/users', {
            headers: { 'X-Session-ID': sessionId }
        });

        if (response.ok) {
            const data = await response.json();
            displayUsers(data.users || []);
        } else {
            document.getElementById('usersList').innerHTML = '<p>Failed to load users</p>';
        }
    } catch (error) {
        document.getElementById('usersList').innerHTML = '<p>Connection error</p>';
    }
}

function displayUsers(users) {
    const container = document.getElementById('usersList');

    if (users.length === 0) {
        container.innerHTML = '<p style="color: #666;">No users registered yet</p>';
        return;
    }

    let html = '<table style="width: 100%; border-collapse: collapse; font-size: 14px;">';
    html += '<tr style="background: #f5f5f5; text-align: left;">';
    html += '<th style="padding: 10px;">Name</th><th>IPs</th><th>Max IPs</th><th>Expires</th><th>Status</th><th>Usage</th><th>Actions</th>';
    html += '</tr>';

    users.forEach(user => {
        const isExpired = new Date(user.expires_at) < new Date();
        const statusColor = user.is_active && !isExpired ? '#27ae60' : '#e74c3c';
        const statusText = user.is_active && !isExpired ? 'Active' : 'Inactive';
        const expiryDate = new Date(user.expires_at).toLocaleDateString();

        // Format IPs list
        const ips = user.ips || [];
        const ipsText = ips.length > 0 ? ips.join('<br>') : '<em style="color: #999;">No IPs yet</em>';
        const ipCount = `${ips.length} / ${user.max_ips}`;

        html += `<tr style="border-bottom: 1px solid #eee;">
            <td style="padding: 10px;"><strong>${user.name}</strong></td>
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
}

async function extendUser(userId) {
    const days = prompt('Extend by how many days?', '30');
    if (!days || isNaN(days)) return;

    try {
        const response = await fetch('/panel/api/users/extend', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Session-ID': sessionId
            },
            body: JSON.stringify({ user_id: userId, days: parseInt(days) })
        });

        if (response.ok) {
            alert('User extended successfully');
            await loadUsers();
        } else {
            const data = await response.json();
            alert(data.error || 'Failed to extend user');
        }
    } catch (error) {
        alert('Connection error');
    }
}

async function deleteUser(userId) {
    if (!confirm('Delete this user?')) return;

    try {
        const response = await fetch('/panel/api/users/delete', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Session-ID': sessionId
            },
            body: JSON.stringify({ user_id: userId })
        });

        if (response.ok) {
            alert('User deleted successfully');
            await loadUsers();
        } else {
            const data = await response.json();
            alert(data.error || 'Failed to delete user');
        }
    } catch (error) {
        alert('Connection error');
    }
}

// ========== Dialog Functions ==========

function showCreateUserDialog() {
    document.getElementById('createUserDialog').style.display = 'flex';
}

function closeCreateUserDialog() {
    document.getElementById('createUserDialog').style.display = 'none';
    document.getElementById('userName').value = '';
    document.getElementById('userDescription').value = '';
    document.getElementById('userMaxIPs').value = '3';
    document.getElementById('userValidDays').value = '30';
}

async function createUser() {
    const name = document.getElementById('userName').value.trim();
    const description = document.getElementById('userDescription').value.trim();
    const maxIPs = parseInt(document.getElementById('userMaxIPs').value);
    const validDays = parseInt(document.getElementById('userValidDays').value);

    console.log('Creating user:', { name, description, maxIPs, validDays });

    if (!name) {
        alert('❌ Please enter a name');
        return;
    }

    if (isNaN(maxIPs) || maxIPs < 1 || maxIPs > 100) {
        alert('❌ Please enter valid max IPs (1-100)');
        return;
    }

    if (isNaN(validDays) || validDays < 1 || validDays > 365) {
        alert('❌ Please enter valid days (1-365)');
        return;
    }

    try {
        const response = await fetch('/panel/api/users/create', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Session-ID': sessionId
            },
            body: JSON.stringify({
                name: name,
                description: description,
                max_ips: maxIPs,
                valid_days: validDays
            })
        });

        if (response.ok) {
            const data = await response.json();
            console.log('User created successfully:', data);

            // Show result dialog
            document.getElementById('userLinkResult').value = data.register_url;
            document.getElementById('resultUserName').textContent = data.user.name;
            document.getElementById('resultUserMaxIPs').textContent = data.user.max_ips;
            document.getElementById('resultUserValidDays').textContent = validDays;
            document.getElementById('resultUserExpires').textContent = new Date(data.user.expires_at).toLocaleDateString();

            closeCreateUserDialog();
            document.getElementById('userResultDialog').style.display = 'flex';
            await loadUsers();
        } else {
            const errorText = await response.text();
            console.error('Error response:', response.status, errorText);
            try {
                const data = JSON.parse(errorText);
                alert('❌ ' + (data.error || 'Failed to create user'));
            } catch (e) {
                alert('❌ Server error: ' + errorText);
            }
        }
    } catch (error) {
        console.error('Exception:', error);
        alert('❌ Connection error: ' + error.message);
    }
}

function copyUserLink() {
    const linkInput = document.getElementById('userLinkResult');
    linkInput.select();
    navigator.clipboard.writeText(linkInput.value);
    alert('✅ Link copied to clipboard!');
}

function closeUserResultDialog() {
    document.getElementById('userResultDialog').style.display = 'none';
}
