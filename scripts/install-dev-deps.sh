#!/bin/bash
# Install local penguin-libs packages for development
#
# This script installs penguin-libs Python packages in editable mode from the local
# penguin-libs clone. These packages are referenced in requirements.in files but
# not yet published to PyPI, so they must be installed from source for development.
#
# Usage: ./scripts/install-dev-deps.sh

set -e

echo "Installing penguin-libs packages for development..."

# Verify penguin-libs clone exists
if [ ! -d ~/code/penguin-libs ]; then
    echo "Error: ~/code/penguin-libs not found. Please clone penguin-libs first:"
    echo "  git clone https://github.com/penguintechinc/penguin-libs ~/code/penguin-libs"
    exit 1
fi

# Install each penguin package in editable mode
packages=(
    "penguin-licensing"
    "penguin-aaa"
    "penguin-limiter"
    "penguin-utils"
    "penguin-pytest"
)

for pkg in "${packages[@]}"; do
    pkg_path="${HOME}/code/penguin-libs/packages/python/${pkg}"
    if [ -d "$pkg_path" ]; then
        echo "Installing ${pkg}..."
        pip3 install -e "$pkg_path"
    else
        echo "Warning: ${pkg_path} not found, skipping"
    fi
done

echo "Done! penguin-libs packages installed."
