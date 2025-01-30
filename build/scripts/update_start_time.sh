#!/bin/bash

YAML_FILES=("./configs/tss1.yaml" "./configs/tss2.yaml" "./configs/tss3.yaml")

if [[ "$OSTYPE" == "darwin"* ]]; then
    NEW_TIME=$(date -u -v+15S +"%Y-%m-%d %H:%M:%S")
else
    NEW_TIME=$(date -u -d "+15 seconds" +"%Y-%m-%d %H:%M:%S")
fi

echo "Updating start_time to: $NEW_TIME"

for file in "${YAML_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "Updating $file..."
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s|start_time: \".*\"|start_time: \"$NEW_TIME\"|" "$file"
        else
            sed -i "s|start_time: \".*\"|start_time: \"$NEW_TIME\"|" "$file"
        fi
        echo "$file updated."
    else
        echo "File $file not found!"
    fi
done

echo "All files updated successfully."