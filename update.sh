#!/bin/bash

# Usage: ./update.sh v1.2.3
# Tags all submodules in this repo with the same version and updates their go.mod requires

set -euo pipefail

if [ -z "${1-}" ]; then
    echo "Usage: $0 vX.Y.Z"
    exit 1
fi

VERSION="$1"

echo "==> Locating all submodules..."

# Find all submodules (subdirs with go.mod, skipping root)
MODULES=($(find . -mindepth 2 -type f -name "go.mod" | sed 's|/go.mod||' | sed 's|^\./||'))

# 1. Tidy, stage and commit any go.mod/go.sum changes
for MOD in "${MODULES[@]}"; do
    echo "==> Tidying $MOD"
    (cd "$MOD" && go mod tidy)
    git add "$MOD/go.mod" "$MOD/go.sum" 2>/dev/null || true
done

# Commit tidy changes if needed
if ! git diff --cached --quiet; then
    git commit -m "chore: go mod tidy before v$VERSION"
fi

# 2. Update cross-submodule require versions and stage changes
for MOD in "${MODULES[@]}"; do
    MOD_PATH="github.com/logistics-id/engine/$MOD"
    for OTHER in "${MODULES[@]}"; do
        if [ "$MOD" != "$OTHER" ]; then
            sed -i.bak -E "s|($MOD_PATH )v[0-9A-Za-z.\-]+|\\1$VERSION|g" "$OTHER/go.mod"
            rm "$OTHER/go.mod.bak"
            git add "$OTHER/go.mod" 2>/dev/null || true
        fi
    done
done

# Commit version bump if needed
if ! git diff --cached --quiet; then
    git commit -m "chore: bump submodule require versions to $VERSION"
fi

# 3. Tag all modules
for MOD in "${MODULES[@]}"; do
    TAG="$MOD/$VERSION"
    if git rev-parse "$TAG" >/dev/null 2>&1; then
        echo "Tag $TAG already exists, skipping."
    else
        git tag "$TAG"
        echo "Tagged $TAG"
    fi
done

# 4. Push tags
echo "==> Pushing all tags to origin..."
git push --tags

echo "==> All submodules tidied, require versions bumped, and tagged with $VERSION!"