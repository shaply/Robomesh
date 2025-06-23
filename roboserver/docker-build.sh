#!/bin/bash

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
docker build -t roboserver-build .
docker create --name temp roboserver-build
docker cp temp:/app/roboserver ./build/roboserver
docker rm temp
scp ./build/roboserver $SERVER_USER@$SERVER_IP:~/roboserver_temp/