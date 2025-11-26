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
OUTPUT_FILE="${IMAGE_NAME}-${VERSION}.tar"

if ! docker buildx inspect "${BUILDER_NAME}" &>/dev/null; then
  echo "Creating buildx builder '${BUILDER_NAME}'..."
  docker buildx create --name "${BUILDER_NAME}" --use
else
  docker buildx use "${BUILDER_NAME}"
fi

echo "Building multi-arch image for linux/amd64,linux/arm64..."
echo "Version: ${VERSION}, Commit: ${COMMIT}"

docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION="${VERSION}" \
  --build-arg COMMIT="${COMMIT}" \
  -t "${IMAGE_NAME}:${VERSION}" \
  --output type=oci,dest="${OUTPUT_FILE}" \
  .

echo ""
echo "Done! Exported to: ${OUTPUT_FILE}"
echo "Size: $(du -h "${OUTPUT_FILE}" | cut -f1)"
echo ""
echo "To load: docker load < ${OUTPUT_FILE}"
