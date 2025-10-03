#!/usr/bin/env python3

import paramiko
import os

# Server details
hostname = "89.47.113.135"
username = "root"
password = "Mefe160502@136525"
remote_path = "/root/smartSNI"

# Files to upload
files = [
    "main.go",
    "go.mod",
    "webpanel.html",
    "config.json",
    "install.sh"
]

print("🚀 Connecting to server...")

# Create SSH client
ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())

try:
    ssh.connect(hostname, username=username, password=password)
    sftp = ssh.open_sftp()

    print("✅ Connected successfully!")

    # Upload files
    for filename in files:
        local_file = f"/home/mehdi/DOH/smartSNI/{filename}"
        remote_file = f"{remote_path}/{filename}"

        if os.path.exists(local_file):
            print(f"📤 Uploading {filename}...")
            sftp.put(local_file, remote_file)
            print(f"✅ {filename} uploaded")
        else:
            print(f"⚠️  {filename} not found locally")

    sftp.close()

    # Rebuild on server
    print("\n🔨 Building on server...")
    stdin, stdout, stderr = ssh.exec_command(
        f"cd {remote_path} && rm -f smartsni && /usr/local/go/bin/go build -o smartsni main.go"
    )
    build_output = stdout.read().decode()
    build_error = stderr.read().decode()

    if build_error:
        print(f"Build output: {build_error}")
    else:
        print("✅ Build successful!")

    # Restart service
    print("\n🔄 Restarting service...")
    stdin, stdout, stderr = ssh.exec_command("systemctl restart sni.service")
    stdout.read()

    # Check status
    print("\n📊 Checking service status...")
    stdin, stdout, stderr = ssh.exec_command("sleep 2 && systemctl status sni.service --no-pager -l")
    status = stdout.read().decode()
    print(status)

    # Check if web panel is running
    print("\n🔍 Checking Web Panel port...")
    stdin, stdout, stderr = ssh.exec_command("ss -tulnp | grep 8088")
    port_check = stdout.read().decode()

    if "8088" in port_check:
        print("✅ Web Panel is running on port 8088!")
        print(f"\n🌐 Web Panel URL: http://{hostname}:8088/panel")
        print("   Username: admin")
        print("   Password: admin (SHA256: 8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918)")
    else:
        print("⚠️  Web Panel port 8088 not found")
        print("\n📝 Last 30 log lines:")
        stdin, stdout, stderr = ssh.exec_command("journalctl -u sni.service -n 30 --no-pager")
        logs = stdout.read().decode()
        print(logs)

    ssh.close()
    print("\n✅ Deployment complete!")

except Exception as e:
    print(f"❌ Error: {e}")
    import traceback
    traceback.print_exc()
