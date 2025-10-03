#!/bin/bash

# Deploy script for Smart SNI Proxy
# Uploads files to remote server and rebuilds the service

SERVER="root@89.47.113.135"
REMOTE_PATH="/root/smartSNI"

echo "🚀 Deploying Smart SNI Proxy to server..."

# Upload main files
echo "📤 Uploading main.go..."
scp main.go $SERVER:$REMOTE_PATH/

echo "📤 Uploading go.mod..."
scp go.mod $SERVER:$REMOTE_PATH/

echo "📤 Uploading webpanel.html..."
scp webpanel.html $SERVER:$REMOTE_PATH/

echo "📤 Uploading config.json..."
scp config.json $SERVER:$REMOTE_PATH/

echo "📤 Uploading install.sh..."
scp install.sh $SERVER:$REMOTE_PATH/

# Rebuild and restart on server
echo "🔨 Building on server..."
ssh $SERVER "cd $REMOTE_PATH && rm -f smartsni && /usr/local/go/bin/go build -o smartsni main.go"

echo "🔄 Restarting service..."
ssh $SERVER "systemctl restart sni.service"

echo "✅ Deployment complete!"

echo "📊 Checking service status..."
ssh $SERVER "systemctl status sni.service --no-pager -l"

echo ""
echo "🌐 Web Panel: http://89.47.113.135:8088/panel"
echo "   Username: admin"
echo "   Password: admin (default - change via config.json)"
