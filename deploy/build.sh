#!/bin/bash
set -e

# Define Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=======================================${NC}"
echo -e "${BLUE}   Wiretify Local Build Script         ${NC}"
echo -e "${BLUE}=======================================${NC}"

# Move to project root
cd ..

echo -e "${GREEN}[+] Compiling for Linux amd64...${NC}"
GO_CMD="/home/accnet/local-go/bin/go"
if [ ! -x "$GO_CMD" ]; then
    GO_CMD="go"
fi
GOOS=linux GOARCH=amd64 $GO_CMD build -ldflags="-s -w" -o deploy/wiretify cmd/server/main.go

echo -e "${GREEN}[+] Copying frontend assets...${NC}"
rm -rf deploy/web
cp -r web deploy/web

echo -e "${BLUE}=======================================${NC}"
echo -e "${GREEN}Build successful!${NC}"
echo -e "Ready for deployment! Instructions:"
echo -e "1. Zip or copy the entire 'deploy' folder to your VPS."
echo -e "2. On the VPS, enter the folder: ${BLUE}cd deploy${NC}"
echo -e "3. Run the installer: ${BLUE}sudo ./install.sh${NC}"
echo -e "${BLUE}=======================================${NC}"
