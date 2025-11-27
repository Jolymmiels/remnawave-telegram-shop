#!/usr/bin/env bash

set -euo pipefail

IMAGE_NAME="private-remnawave-telegram-shop-bot"
BUILDER_NAME="multiarch-builder"

read -rp "Enter version (e.g. 3.4.6): " VERSION
if [[ -z "${VERSION}" ]]; then
  echo "Version must not be empty" >&2
  exit 1
fi

COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "none")
OUTPUT_AMD64="${IMAGE_NAME}-${VERSION}-amd64.tar"
OUTPUT_ARM64="${IMAGE_NAME}-${VERSION}-arm64.tar"

if ! docker buildx inspect "${BUILDER_NAME}" &>/dev/null; then
  echo "Creating buildx builder '${BUILDER_NAME}'..."
  docker buildx create --name "${BUILDER_NAME}" --use
else
  docker buildx use "${BUILDER_NAME}"
fi

echo "Version: ${VERSION}, Commit: ${COMMIT}"
echo ""

echo "Building linux/amd64..."
docker buildx build \
  --platform linux/amd64 \
  --build-arg VERSION="${VERSION}" \
  --build-arg COMMIT="${COMMIT}" \
  -t "${IMAGE_NAME}:${VERSION}" \
  --output type=docker,dest="${OUTPUT_AMD64}" \
  .

echo ""
echo "Building linux/arm64..."
docker buildx build \
  --platform linux/arm64 \
  --build-arg VERSION="${VERSION}" \
  --build-arg COMMIT="${COMMIT}" \
  -t "${IMAGE_NAME}:${VERSION}" \
  --output type=docker,dest="${OUTPUT_ARM64}" \
  .

echo ""
echo "Done!"
echo "  AMD64: ${OUTPUT_AMD64} ($(du -h "${OUTPUT_AMD64}" | cut -f1))"
echo "  ARM64: ${OUTPUT_ARM64} ($(du -h "${OUTPUT_ARM64}" | cut -f1))"
echo ""
echo "To load:"
echo "  docker load -i ${OUTPUT_AMD64}"
echo "  docker load -i ${OUTPUT_ARM64}"
