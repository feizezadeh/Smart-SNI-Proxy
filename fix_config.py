#!/usr/bin/env python3

import paramiko

# Server details
hostname = "89.47.113.135"
username = "root"
password = "Mefe160502@136525"

print("🔧 Fixing config.json on server...")

ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())

try:
    ssh.connect(hostname, username=username, password=password)

    # Get server IP
    stdin, stdout, stderr = ssh.exec_command("hostname -I | awk '{print $1}'")
    server_ip = stdout.read().decode().strip()
    print(f"📍 Server IP: {server_ip}")

    # Fix config.json - replace <YOUR_IP> and <YOUR_HOST>
    fix_command = f"""
    cd /root/smartSNI
    sed -i 's/<YOUR_IP>/{server_ip}/g' config.json
    sed -i 's/<YOUR_HOST>/your-domain.com/g' config.json
    cat config.json
    """

    stdin, stdout, stderr = ssh.exec_command(fix_command)
    output = stdout.read().decode()
    print("\n✅ Fixed config.json:")
    print(output)

    # Restart service
    print("\n🔄 Restarting service...")
    stdin, stdout, stderr = ssh.exec_command("systemctl restart sni.service && sleep 3")
    stdout.read()

    # Check status
    print("\n📊 Service status:")
    stdin, stdout, stderr = ssh.exec_command("systemctl status sni.service --no-pager -l | head -20")
    print(stdout.read().decode())

    # Check web panel
    print("\n🔍 Checking Web Panel...")
    stdin, stdout, stderr = ssh.exec_command("ss -tulnp | grep 8088")
    port_check = stdout.read().decode()

    if "8088" in port_check:
        print("✅ Web Panel is running on port 8088!")
        print(f"\n🌐 Access at: http://{hostname}:8088/panel")
    else:
        print("⚠️  Web Panel not running. Checking logs...")
        stdin, stdout, stderr = ssh.exec_command("journalctl -u sni.service -n 20 --no-pager")
        print(stdout.read().decode())

    ssh.close()

except Exception as e:
    print(f"❌ Error: {e}")
    import traceback
    traceback.print_exc()
