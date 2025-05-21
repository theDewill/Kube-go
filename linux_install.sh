#!/bin/bash

# Install Wails
echo "Installing Wails framework..."
go install github.com/wailsapp/wails/v2/cmd/wails@latest
if [ $? -ne 0 ]; then
    echo "Failed to install Wails framework"
    exit 1
fi
echo "Wails installed successfully!"

# Create base directory based on XDG spec
HOME_DIR="$HOME"
XDG_DATA_HOME="${XDG_DATA_HOME:-$HOME_DIR/.local/share}"
BASE_DIR="$XDG_DATA_HOME/Kube"
mkdir -p "$BASE_DIR"
if [ $? -ne 0 ]; then
    echo "Failed to create base directory: $BASE_DIR"
    exit 1
fi

# Create models subdirectory
MODELS_DIR="$BASE_DIR/models"
mkdir -p "$MODELS_DIR"
if [ $? -ne 0 ]; then
    echo "Failed to create models directory: $MODELS_DIR"
    exit 1
fi

# Get script directory to locate f_model
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_FILE="$SCRIPT_DIR/f_model/test.xml"

# Copy test.xml file to models directory
cp "$SOURCE_FILE" "$MODELS_DIR/"
if [ $? -ne 0 ]; then
    echo "Failed to copy $SOURCE_FILE to $MODELS_DIR"
    exit 1
fi

echo "Installation completed successfully!"
echo "Base directory: $BASE_DIR"
echo "Models directory: $MODELS_DIR"
