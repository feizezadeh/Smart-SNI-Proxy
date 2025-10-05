#!/bin/bash

# اسکریپت بهینه‌سازی سرور - فقط تنظیمات سیستم
# بدون نصب هیچ برنامه‌ای

echo "🔧 شروع بهینه‌سازی سیستم عامل..."
echo ""

# ======================================
# 1. تنظیمات شبکه (sysctl)
# ======================================
echo "🌐 بهینه‌سازی تنظیمات شبکه..."

cat > /etc/sysctl.d/99-smartsni-optimize.conf << 'EOF'
# Network Performance Tuning
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_rmem = 4096 87380 67108864
net.ipv4.tcp_wmem = 4096 65536 67108864
net.core.netdev_max_backlog = 5000

# TCP BBR Congestion Control
net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr

# TCP Performance
net.ipv4.tcp_notsent_lowat = 16384
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_fastopen = 3
net.ipv4.tcp_mtu_probing = 1

# Connection Tuning
net.ipv4.tcp_max_syn_backlog = 8192
net.ipv4.tcp_tw_reuse = 1
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.tcp_fin_timeout = 15

# TCP Keepalive
net.ipv4.tcp_keepalive_time = 300
net.ipv4.tcp_keepalive_probes = 3
net.ipv4.tcp_keepalive_intvl = 30

# File Descriptors
fs.file-max = 2097152
fs.nr_open = 2097152

# Memory Management
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5

# Security
net.ipv4.conf.default.rp_filter = 1
net.ipv4.conf.all.rp_filter = 1
net.ipv4.tcp_syncookies = 1
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.all.accept_source_route = 0
EOF

# اعمال تنظیمات
sysctl -p /etc/sysctl.d/99-smartsni-optimize.conf > /dev/null 2>&1

echo "   ✅ TCP BBR فعال شد"
echo "   ✅ TCP buffers بهینه شد"
echo "   ✅ File descriptors افزایش یافت"

# ======================================
# 2. تنظیمات ulimit
# ======================================
echo ""
echo "📂 تنظیم File Descriptors Limits..."

# حذف تنظیمات قبلی
sed -i '/smartsni/d' /etc/security/limits.conf 2>/dev/null
sed -i '/^root.*nofile/d' /etc/security/limits.conf 2>/dev/null
sed -i '/^\*.*nofile/d' /etc/security/limits.conf 2>/dev/null

cat >> /etc/security/limits.conf << 'EOF'

# SmartSNI Optimization
root soft nofile 1048576
root hard nofile 1048576
root soft nproc 512
root hard nproc 512

* soft nofile 65535
* hard nofile 65535
* soft nproc 4096
* hard nproc 4096
EOF

echo "   ✅ ulimits تنظیم شد (1048576)"

# ======================================
# 3. تنظیمات systemd
# ======================================
echo ""
echo "⚙️  تنظیم systemd defaults..."

mkdir -p /etc/systemd/system.conf.d/
cat > /etc/systemd/system.conf.d/limits.conf << 'EOF'
[Manager]
DefaultLimitNOFILE=1048576
DefaultLimitNPROC=512
EOF

systemctl daemon-reexec

echo "   ✅ systemd limits افزایش یافت"

# ======================================
# 4. تنظیمات I/O Scheduler
# ======================================
echo ""
echo "💾 بهینه‌سازی I/O Scheduler..."

# برای SSD از none/noop استفاده کن
if [ -b /dev/sda ]; then
    echo none > /sys/block/sda/queue/scheduler 2>/dev/null || \
    echo noop > /sys/block/sda/queue/scheduler 2>/dev/null
    echo "   ✅ I/O Scheduler: $(cat /sys/block/sda/queue/scheduler 2>/dev/null || echo 'default')"
fi

# ======================================
# 5. غیرفعال کردن IPv6 (اختیاری)
# ======================================
echo ""
echo "🔧 پیکربندی IPv6..."

read -p "غیرفعال کردن IPv6؟ (y/n) [default: n]: " disable_ipv6
disable_ipv6=${disable_ipv6:-n}

if [[ "$disable_ipv6" == "y" || "$disable_ipv6" == "Y" ]]; then
    cat >> /etc/sysctl.d/99-smartsni-optimize.conf << 'EOF'

# Disable IPv6
net.ipv6.conf.all.disable_ipv6 = 1
net.ipv6.conf.default.disable_ipv6 = 1
net.ipv6.conf.lo.disable_ipv6 = 1
EOF
    sysctl -p /etc/sysctl.d/99-smartsni-optimize.conf > /dev/null 2>&1
    echo "   ✅ IPv6 غیرفعال شد"
else
    echo "   ℹ️  IPv6 فعال باقی ماند"
fi

# ======================================
# 6. تنظیمات Transparent Huge Pages
# ======================================
echo ""
echo "💾 تنظیم Transparent Huge Pages..."

echo never > /sys/kernel/mm/transparent_hugepage/enabled 2>/dev/null
echo never > /sys/kernel/mm/transparent_hugepage/defrag 2>/dev/null

# دائمی کردن در boot
cat > /etc/systemd/system/disable-thp.service << 'EOF'
[Unit]
Description=Disable Transparent Huge Pages
DefaultDependencies=no
After=sysinit.target local-fs.target
Before=basic.target

[Service]
Type=oneshot
ExecStart=/bin/sh -c 'echo never > /sys/kernel/mm/transparent_hugepage/enabled'
ExecStart=/bin/sh -c 'echo never > /sys/kernel/mm/transparent_hugepage/defrag'

[Install]
WantedBy=basic.target
EOF

systemctl daemon-reload
systemctl enable disable-thp.service > /dev/null 2>&1

echo "   ✅ THP غیرفعال شد"

# ======================================
# 7. تنظیمات Time
# ======================================
echo ""
echo "🕐 همگام‌سازی زمان..."

timedatectl set-ntp true > /dev/null 2>&1
echo "   ✅ NTP فعال شد"

# ======================================
# 8. نمایش خلاصه
# ======================================
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ بهینه‌سازی کامل شد!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📊 تنظیمات اعمال شده:"
echo ""
echo "🌐 Network:"
echo "   • TCP Congestion: $(sysctl -n net.ipv4.tcp_congestion_control)"
echo "   • TCP Fast Open: $(sysctl -n net.ipv4.tcp_fastopen)"
echo "   • Port Range: $(sysctl -n net.ipv4.ip_local_port_range)"
echo "   • Max Backlog: $(sysctl -n net.core.netdev_max_backlog)"
echo ""
echo "📂 File Descriptors:"
echo "   • System Max: $(sysctl -n fs.file-max)"
echo "   • Open Files: $(sysctl -n fs.nr_open)"
echo ""
echo "💾 Memory:"
echo "   • Swappiness: $(sysctl -n vm.swappiness)"
echo "   • Dirty Ratio: $(sysctl -n vm.dirty_ratio)%"
echo ""
echo "🔐 Security:"
echo "   • SYN Cookies: $(sysctl -n net.ipv4.tcp_syncookies)"
echo "   • RP Filter: $(sysctl -n net.ipv4.conf.all.rp_filter)"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "⚠️  توصیه‌ها:"
echo "   1. سیستم را reboot کنید برای اعمال کامل تغییرات"
echo "   2. بعد از reboot، تنظیمات را با دستور زیر چک کنید:"
echo "      sysctl net.ipv4.tcp_congestion_control"
echo "      ulimit -n"
echo ""
echo "📝 برای reboot:"
echo "   reboot"
echo ""
