#!/bin/bash

# Script purpose: List the file structure of the current directory and save it to a file (alternative to the 'tree' command).
# Ignores .git directories and cache files.

# Define output file name
output_file="directory_structure.txt"

# Get current directory
current_dir=$(pwd)

# Define exclusion patterns
exclude_patterns=(
    "/.git"          # .git directory
    "*~"             # Backup files (e.g., *.swp, *~)
    "#*#"            # Temporary files (e.g., .#*~)
    "._*"            # Hidden files (e.g., .*~)
    "$output_file"   # The output file itself
    "*.pyc"          # Python cache files
)

# List file and folder structure in the current directory, save to file
echo "Directory structure ($current_dir):" > "$output_file"

# Find files and directories, excluding patterns, and format output
find "$current_dir" -print | while read -r file; do
    for pattern in "${exclude_patterns[@]}"; do
        if [[ "$file" == *"$pattern"* ]]; then
            continue 2  # Skip to the next file in the outer loop
        fi
    done

    # Format file path with pipe characters
    formatted_path=$(echo "$file" | sed -e 's;[^/]*/;|____;g;s;____|; |;g')
    echo "$formatted_path" >> "$output_file" 
done

# Display results to standard output
echo "Directory structure saved to $output_file. Contents:"
cat "$output_file"
