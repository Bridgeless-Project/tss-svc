#!/bin/bash

YAML_FILE="./docker-compose.yaml"

# Check if the new command is provided as a parameter
if [ -z "$1" ]; then
    echo "Error: New command value is required as a parameter."
    echo "Usage: $0 <new_command>"
    exit 1
fi

NEW_COMMAND="$1" # Get the first parameter

echo "Replacing 'tss-svc service run' in $YAML_FILE with: $NEW_COMMAND"

# Check if the file exists
if [ -f "$YAML_FILE" ]; then
    echo "Updating $YAML_FILE..."
    # Use sed to update the file
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s|tss-svc service run.*|$NEW_COMMAND|" "$YAML_FILE"
    else
        sed -i "s|tss-svc service run.*|$NEW_COMMAND|" "$YAML_FILE"
    fi
    echo "$YAML_FILE updated."
else
    echo "File $YAML_FILE not found!"
fi

echo "File update process complete."