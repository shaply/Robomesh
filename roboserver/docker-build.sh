#!/bin/bash

# filepath: /Users/davidgu/Desktop/Projects/Robomesh/roboserver/docker-build.sh

if [ -f .env ]; then
    echo "Loading environment variables from .env file..."
    export $(grep -v '^#' .env | xargs)
else
    echo ".env file not found. Please ensure it exists."
    exit 1
fi

echo "Building roboserver for $SERVER_USER@$SERVER_IP..."
rm -rf ./build
mkdir -p ./build

# Build with error checking
echo "Building Docker image..."
if ! docker build -t roboserver-build .; then
    echo "ERROR: Docker build failed!"
    exit 1
fi

echo "Extracting binary from container..."
if ! docker create --name temp roboserver-build; then
    echo "ERROR: Failed to create container!"
    exit 1
fi

if ! docker cp temp:/app/roboserver ./build/roboserver; then
    echo "ERROR: Failed to copy binary from container!"
    docker rm temp 2>/dev/null
    exit 1
fi

if ! docker rm temp; then
    echo "WARNING: Failed to remove temporary container"
fi

echo "Uploading to server..."
if ! scp ./build/roboserver $SERVER_USER@$SERVER_IP:~/roboserver_temp/; then
    echo "ERROR: Failed to upload binary to server!"
    exit 1
fi

echo "Build and upload completed successfully!"