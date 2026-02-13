#!/bin/bash
# MarchProxy OBS Plugin Installer for macOS

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_NAME="marchproxy-stream.lua"
OBS_SCRIPTS_DIR="$HOME/Library/Application Support/obs-studio/scripts"

echo "MarchProxy OBS Plugin Installer"
echo "================================"
echo

# Check if OBS is installed
if ! [ -d "/Applications/OBS.app" ] && ! [ -d "$HOME/Applications/OBS.app" ]; then
    echo "Warning: OBS Studio doesn't appear to be installed."
    echo "The script will be copied anyway, but you'll need OBS to use it."
    echo
fi

# Create scripts directory if it doesn't exist
if [ ! -d "$OBS_SCRIPTS_DIR" ]; then
    echo "Creating OBS scripts directory..."
    mkdir -p "$OBS_SCRIPTS_DIR"
fi

# Copy the script
echo "Installing $SCRIPT_NAME to $OBS_SCRIPTS_DIR..."
cp "$SCRIPT_DIR/$SCRIPT_NAME" "$OBS_SCRIPTS_DIR/"

# Verify installation
if [ -f "$OBS_SCRIPTS_DIR/$SCRIPT_NAME" ]; then
    echo
    echo "Installation successful!"
    echo
    echo "To enable the plugin:"
    echo "1. Open OBS Studio"
    echo "2. Go to Tools -> Scripts"
    echo "3. Click '+' and select '$SCRIPT_NAME'"
    echo "4. Configure your MarchProxy settings"
    echo
else
    echo "Error: Installation failed"
    exit 1
fi
