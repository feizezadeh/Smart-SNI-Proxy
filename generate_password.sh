#!/bin/bash

# Generate Password Hash for Web Panel
# Usage: ./generate_password.sh [password]

if [ -z "$1" ]; then
    read -sp "Enter password: " password
    echo
else
    password="$1"
fi

hash=$(echo -n "$password" | sha256sum | awk '{print $1}')

echo ""
echo "========================================="
echo "Password Hash Generator"
echo "========================================="
echo ""
echo "Password: $password"
echo "SHA256 Hash: $hash"
echo ""
echo "Add this hash to config.json:"
echo "\"web_panel_password\": \"$hash\""
echo ""
echo "========================================="
