#!/bin/bash

TEMPLATE_FILE="./assets/templates/module-info.md"
if [ ! -f "$TEMPLATE_FILE" ]; then
    echo "Template file '$TEMPLATE_FILE' not found. Exiting."
    exit 1
fi

# Check if a folder path is provided
if [ -z "$1" ]; then
    echo "Folder path not provided"
    exit 1
fi

# Get the folder path and last folder name
FOLDER_PATH=$1
if [ ! -d "$FOLDER_PATH" ]; then
    echo "Error: '$FOLDER_PATH' is not a valid directory."
    exit 1
fi

README_FILE="$FOLDER_PATH/README.md"
cp "$TEMPLATE_FILE" "$README_FILE"
