#!/bin/bash
# Script to build and run Wiretify in development mode

echo "Building Wiretify..."
/home/accnet/local-go/bin/go build -o wiretify cmd/server/main.go

if [ $? -eq 0 ]; then
    echo "Build successful. Starting server with sudo..."
    sudo ./wiretify
else
    echo "Build failed!"
    exit 1
fi
