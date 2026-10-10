#!/bin/bash

# BusinessOS Workspace Container Build Script
set -e

IMAGE_NAME="businessos-workspace"
IMAGE_TAG="latest"
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"

echo "Building workspace container image..."
echo "Image: ${FULL_IMAGE}"
echo ""

# Build the Docker image
docker build \
  --tag "${FULL_IMAGE}" \
  --file Dockerfile \
  .

echo ""
echo "Build complete!"
echo ""
echo "Image details:"
docker images "${IMAGE_NAME}" | head -2

echo ""
echo "To run the workspace container:"
echo "  docker run -it --rm -v \$(pwd):/workspace ${FULL_IMAGE}"
echo ""
echo "To run with network access:"
echo "  docker run -it --rm --network host -v \$(pwd):/workspace ${FULL_IMAGE}"
