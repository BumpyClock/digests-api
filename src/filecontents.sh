#!/bin/bash

# Check if an argument is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <file-extension>"
    exit 1
fi

# File extension from the command line argument
file_extension=$1

# Output file name
output_file="concatenated_output.txt"

# Clear the output file or create it if it doesn't exist
> $output_file

# Find all files with the given extension and process each one
find . -type f -name "*.$file_extension" | while read filename; do
    echo "//$(basename "$filename")" >> $output_file
    cat "$filename" >> $output_file
    echo "" >> $output_file # Optional: Adds a newline for separation
done

echo "All files have been concatenated into $output_file"