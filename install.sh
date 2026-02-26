#!/bin/bash
set -e

# Define Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=======================================${NC}"
echo -e "${BLUE}       Wiretify Installation Script    ${NC}"
echo -e "${BLUE}=======================================${NC}"

# 1. Check Root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Please run as root (use sudo)!${NC}"
  exit 1
fi

# 2. Check and Install WireGuard
echo -e "${GREEN}[+] Checking and installing WireGuard...${NC}"
if [ -x "$(command -v apt-get)" ]; then
    apt-get update -y
    apt-get install -y wireguard iptables iproute2 curl wget
elif [ -x "$(command -v yum)" ]; then
    yum install -y epel-release
    yum install -y wireguard-tools iptables iproute curl wget
else
    echo -e "${RED}Unsupported package manager. Please install wireguard and iptables manually.${NC}"
    exit 1
fi

# Enable IPv4 forwarding in kernel
echo -e "${GREEN}[+] Enabling IPv4 IP Forwarding...${NC}"
sed -i 's/#net.ipv4.ip_forward=1/net.ipv4.ip_forward=1/g' /etc/sysctl.conf
if ! grep -q "net.ipv4.ip_forward=1" /etc/sysctl.conf; then
    echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
fi
sysctl -p > /dev/null || sysctl -w net.ipv4.ip_forward=1 > /dev/null

# 3. Determine Public IP
PUBLIC_IP=$(curl -s ifconfig.me)
if [ -z "$PUBLIC_IP" ]; then
    PUBLIC_IP="127.0.0.1"
fi
echo -e "${GREEN}[+] Detected Public IP: ${PUBLIC_IP}${NC}"

# 4. Prepare deployment directory
echo -e "${GREEN}[+] Setting up /opt/wiretify directory...${NC}"
mkdir -p /opt/wiretify/web
mkdir -p /opt/wiretify/data

# 5. Build Wiretify (Assuming script is run in the project root)
if [ ! -f "cmd/server/main.go" ]; then
    echo -e "${RED}Error: main.go not found! Please run this script from the Wiretify project root directory.${NC}"
    exit 1
fi

echo -e "${GREEN}[+] Building Wiretify binary...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${BLUE}[*] Go is not installed. Attempting to install Go via snap...${NC}"
    snap install go --classic || echo -e "${RED}Failed to install Go. Please install it manually.${NC}"
fi

# Sử dụng variable GO executable
GO_CMD=$(command -v go || echo "/snap/bin/go")
$GO_CMD build -o wiretify cmd/server/main.go

# 6. Copy files to /opt
echo -e "${GREEN}[+] Copying files to /opt/wiretify...${NC}"
cp wiretify /opt/wiretify/
cp -r web/* /opt/wiretify/web/

# 7. Create Systemd Service
echo -e "${GREEN}[+] Creating systemd service...${NC}"
cat <<EOF > /etc/systemd/system/wiretify.service
[Unit]
Description=Wiretify VPN Dashboard
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/wiretify
Environment="WIRETIFY_SERVER_ENDPOINT=${PUBLIC_IP}"
Environment="WIRETIFY_DB_PATH=/opt/wiretify/data/wiretify.db"
ExecStart=/opt/wiretify/wiretify
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 8. Start Service
echo -e "${GREEN}[+] Starting Wiretify service...${NC}"
systemctl daemon-reload
systemctl enable wiretify
systemctl restart wiretify

# 9. Announce
echo -e "${BLUE}=======================================${NC}"
echo -e "${GREEN}Wiretify installed successfully!${NC}"
echo -e "Dashboard: http://${PUBLIC_IP}:8080"
echo -e "WireGuard Port: 51820 (Ensure this UDP port is open in your firewall)"
echo -e "Service Status: systemctl status wiretify"
echo -e "To view logs run: journalctl -fu wiretify"
echo -e "${BLUE}=======================================${NC}"
