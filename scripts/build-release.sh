#!/usr/bin/env bash
# Build and release script for unquarantine
# Usage: ./scripts/build-release.sh <version>
#
# This script:
#   1. Cross-compiles arm64 + amd64 binaries from the current project
#   2. Packages them as tar.gz
#   3. Updates Formula/unquarantine.rb in the homebrew tap with new version + sha256
#
# Run from the unquarantine project root.

set -euo pipefail

# Config
VERSION="${1:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TAP_ROOT="$(cd "${PROJECT_ROOT}/../homebrew-tap" 2>/dev/null || echo '')"
FORMULA_PATH="${TAP_ROOT}/Formula/unquarantine.rb"
DIST_DIR="${PROJECT_ROOT}/dist"
GITHUB_REPO="jianzhoujz/unquarantine"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
GRAY='\033[90m'
NC='\033[0m'

# Validate
if [[ -z "${VERSION}" ]]; then
    echo -e "${RED}Error: missing version${NC}"
    echo "Usage: $0 <version>"
    exit 1
fi

VERSION="${VERSION#v}"

if [[ ! -f "${PROJECT_ROOT}/main.go" ]]; then
    echo -e "${RED}Error: main.go not found in ${PROJECT_ROOT}${NC}"
    exit 1
fi

if [[ -z "${TAP_ROOT}" || ! -f "${FORMULA_PATH}" ]]; then
    echo -e "${RED}Error: homebrew tap not found at ${TAP_ROOT}${NC}"
    echo "Expected: ${FORMULA_PATH}"
    exit 1
fi

echo -e "${CYAN}Building unquarantine v${VERSION}${NC}"
echo -e "${GRAY}Source: ${PROJECT_ROOT}${NC}"
echo -e "${GRAY}Output: ${DIST_DIR}${NC}"
echo -e "${GRAY}Tap:    ${TAP_ROOT}${NC}"

rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

# Build
ARM64_SHA=""
AMD64_SHA=""

build_platform() {
    local GOOS="$1"
    local GOARCH="$2"
    local OUTPUT="unquarantine-${VERSION}-${GOOS}-${GOARCH}"
    local BINARY="${DIST_DIR}/${OUTPUT}"
    local ARCHIVE="${OUTPUT}.tar.gz"

    echo -e "${GRAY}Building ${GOOS}/${GOARCH}...${NC}"

    cd "${PROJECT_ROOT}"
    CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -ldflags="-s -w" -o "${BINARY}" .

    cd "${DIST_DIR}"
    tar -czf "${ARCHIVE}" "${OUTPUT}"

    local SHA
    SHA=$(shasum -a 256 "${ARCHIVE}" | awk '{print $1}')
    rm "${BINARY}"

    echo -e "${GREEN}OK${NC}   ${GOARCH}: ${SHA:0:16}... ${ARCHIVE}"

    if [[ "${GOARCH}" == "arm64" ]]; then
        ARM64_SHA="${SHA}"
    else
        AMD64_SHA="${SHA}"
    fi
}

build_platform darwin arm64
build_platform darwin amd64

# Update Formula
echo -e "${GRAY}Updating Formula...${NC}"

ruby - "${VERSION}" "${ARM64_SHA}" "${AMD64_SHA}" "${FORMULA_PATH}" <<'RUBY'
version, arm64_sha, amd64_sha, path = ARGV

content = File.read(path)
content = content.sub(/version "[^"]+"/, "version \"#{version}\"")

content = content.sub(
  /on_arm do\s+url "[^"]+"\s+sha256 "[^"]+"/,
  "on_arm do\n      url \"https://github.com/jianzhoujz/unquarantine/releases/download/v#{version}/unquarantine-#{version}-darwin-arm64.tar.gz\"\n      sha256 \"#{arm64_sha}\""
)

content = content.sub(
  /on_intel do\s+url "[^"]+"\s+sha256 "[^"]+"/,
  "on_intel do\n      url \"https://github.com/jianzhoujz/unquarantine/releases/download/v#{version}/unquarantine-#{version}-darwin-amd64.tar.gz\"\n      sha256 \"#{amd64_sha}\""
)

File.write(path, content)
RUBY

echo -e "${GREEN}OK${NC}   Formula updated"

# Summary
echo
echo -e "${CYAN}Results:${NC}"
find "${DIST_DIR}" -name "*.tar.gz" -exec ls -lh {} \; | awk '{print "  " $NF ": " $5}'
echo
echo -e "${CYAN}SHA256:${NC}"
echo "  arm64: ${ARM64_SHA}"
echo "  amd64: ${AMD64_SHA}"
echo

# Next steps
cat <<EOF
Next steps:

1. Commit Formula in tap:
   cd "${TAP_ROOT}"
   git add Formula/unquarantine.rb
   git commit -m "unquarantine: update to v${VERSION}"

2. Create tag in this repo:
   cd "${PROJECT_ROOT}"
   git tag "v${VERSION}"
   git push origin "v${VERSION}"

3. Create GitHub Release:
   gh release create "v${VERSION}" \
     "${DIST_DIR}/unquarantine-${VERSION}-darwin-arm64.tar.gz" \
     "${DIST_DIR}/unquarantine-${VERSION}-darwin-amd64.tar.gz" \
     --repo "${GITHUB_REPO}" \
     --title "v${VERSION}" \
     --notes "brew tap jianzhoujz/tap && brew install unquarantine"

4. Push tap:
   cd "${TAP_ROOT}"
   git push origin main
EOF
