#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Update package list and install prerequisites
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common

# Install Docker
echo "Installing Docker..."
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
sudo apt-get update
sudo apt-get install -y docker-ce

# Start Docker and enable it to start at boot
sudo systemctl start docker
sudo systemctl enable docker

# Add current user to the docker group to run docker without sudo
sudo usermod -aG docker ${USER}

# Download and install Go
echo "Installing Go..."
GO_VERSION="1.20.5"
curl -O https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
rm go${GO_VERSION}.linux-amd64.tar.gz

# Set up Go environment variables
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
source ~/.profile

# Verify installations
echo "Verifying Docker installation..."
docker --version

echo "Verifying Go installation..."
go version

echo "Docker and Go installation completed successfully!"
echo "You might need to log out and log back in to apply the Docker group changes."

#docker tag couchbase/observability-stack:v1 pocan101/bbva-persistence-amd64
docker run -v bbva:/prometheus -d -p 8080:8080 --name cmos -e PROMETHEUS_STORAGE_MAX_SIZE=12GB pocan101/cmos:bbva-amd64-persistency-3