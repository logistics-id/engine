#!/bin/bash

# Usage: ./update.sh v1.2.3
# Updates all modules in the monorepo (including root) and versions them correctly
# Including: tidy, replace directives, commit bumps, dual tagging

set -euo pipefail

if [ -z "${1-}" ]; then
    echo "Usage: $0 vX.Y.Z"
    exit 1
fi

VERSION="$1"
ROOT_MODULE="github.com/logistics-id/engine"

echo "==> Locating all modules..."
MODULES=("." $(find . -mindepth 2 -type f -name "go.mod" | sed 's|/go.mod||' | sed 's|^\./||'))

# 1. Add temporary replace directives and tidy
for MOD in "${MODULES[@]}"; do
    echo "==> Tidying $MOD with valid temporary replace directives"

    MOD_GO_MOD="$MOD/go.mod"
    # Parse all require lines (support block and inline)
    REQUIRES=$(sed -n -E 's/^\s*require\s*\(?//p' "$MOD_GO_MOD" | awk '{print $1}')

    for DEP in "${MODULES[@]}"; do
        [ "$MOD" = "$DEP" ] && continue

        DEP_PATH="$ROOT_MODULE"
        [ "$DEP" != "." ] && DEP_PATH+="/$DEP"

        if echo "$REQUIRES" | grep -q "^$DEP_PATH"; then
            REL_PATH=$(python3 -c "import os.path; p=os.path.relpath('$DEP', start='$MOD'); print(p if p.startswith('.') else './' + p)")
            echo "replace $DEP_PATH => $REL_PATH" >> "$MOD_GO_MOD"
        fi
    done

    (cd "$MOD" && go mod tidy)

    # Remove temporary replaces
    sed -i '' '/^replace .* => .*$/d' "$MOD_GO_MOD"

    git add "$MOD_GO_MOD" "$MOD/go.sum" 2>/dev/null || true
done

# Commit tidy changes if needed
if ! git diff --cached --quiet; then
    git commit -m "chore: go mod tidy before $VERSION"
fi

# 2. Bump require versions across all modules
for MOD in "${MODULES[@]}"; do
    MOD_PATH="$ROOT_MODULE"
    [ "$MOD" != "." ] && MOD_PATH+="/$MOD"

    for TARGET in "${MODULES[@]}"; do
        sed -i '' -E "s|($MOD_PATH )v[0-9A-Za-z.\-]+|\\1$VERSION|g" "$TARGET/go.mod" || true
        git add "$TARGET/go.mod" 2>/dev/null || true
    done
done

# Commit version bump if needed
if ! git diff --cached --quiet; then
    git commit -m "chore: bump require versions to $VERSION"
fi

# 3. Tag each module (dual-tag: vX.Y.Z + path/vX.Y.Z)
for MOD in "${MODULES[@]}"; do
    TAG="$VERSION"
    NAMESPACED_TAG="$VERSION"
    [ "$MOD" != "." ] && NAMESPACED_TAG="$MOD/$VERSION"

    cd "$MOD"

    # Create plain vX.Y.Z tag for Go proxy
    if git rev-parse "$TAG" >/dev/null 2>&1; then
        echo "Tag $TAG already exists (proxy-safe), skipping."
    else
        git tag "$TAG"
        echo "Tagged $MOD: $TAG (Go proxy)"
    fi

    # Create path-prefixed tag for human reference
    if git rev-parse "$NAMESPACED_TAG" >/dev/null 2>&1; then
        echo "Tag $NAMESPACED_TAG already exists, skipping."
    else
        git tag "$NAMESPACED_TAG"
        echo "Tagged $MOD: $NAMESPACED_TAG (namespaced)"
    fi

    cd - >/dev/null
done

# 4. Push all tags
echo "==> Pushing all tags to origin..."
git push --tags

echo "✅ All modules tidied, require versions bumped, and tagged (v$VERSION and path/$VERSION)!"