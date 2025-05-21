#!/bin/bash

# Install Wails
echo "Installing Wails framework..."
go install github.com/wailsapp/wails/v2/cmd/wails@latest
if [ $? -ne 0 ]; then
    echo "Failed to install Wails framework"
    exit 1
fi
echo "Wails installed successfully!"

# Create base directory (~/Library/Application Support/Kube)
HOME_DIR="$HOME"
BASE_DIR="$HOME_DIR/Library/Application Support/Kube"
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
