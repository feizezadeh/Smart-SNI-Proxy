#!/bin/bash

# Simple upload script using scp
SERVER="root@89.47.113.135"
REMOTE_DIR="/root/smartSNI"

echo "📤 Uploading files to server..."

# Upload main files
sshpass -p 'Mefe160502@136525' scp main.go $SERVER:$REMOTE_DIR/
sshpass -p 'Mefe160502@136525' scp webpanel.html $SERVER:$REMOTE_DIR/
sshpass -p 'Mefe160502@136525' scp config.json $SERVER:$REMOTE_DIR/

echo "✅ Files uploaded!"
echo ""
echo "Now run these commands on the server:"
echo "cd /root/smartSNI"
echo "/usr/local/go/bin/go build -o smartsni main.go"
echo "sed -i 's/<YOUR_IP>/89.47.113.135/g' config.json"
echo "systemctl restart sni.service"
echo "systemctl status sni.service"
