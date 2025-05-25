#!/bin/bash

# Function to count lines in a directory and its subdirectories
count_lines() {
    local dir=$1
    local maxdepth=$2
    local source_lines=$(find "$dir" -maxdepth "$maxdepth" -name "*.go" ! -name "*_test.go" -type f -exec cat {} \; | wc -l)
    local test_lines=$(find "$dir" -maxdepth "$maxdepth" -name "*_test.go" -type f -exec cat {} \; | wc -l)
    echo "$source_lines $test_lines"
}

# Function to print directory stats
print_stats() {
    local dir=$1
    local indent=$2
    local is_leaf=$3
    
    # Count lines in current directory
    local source_lines test_lines
    if [ "$is_leaf" = "true" ]; then
        # For leaf directories, only count files in current directory
        read source_lines test_lines < <(count_lines "$dir" 1)
    else
        # For parent directories, count all files including subdirectories
        read source_lines test_lines < <(count_lines "$dir" 999)
    fi
    
    # Print directory stats
    printf "%${indent}s%s: %d source lines, %d test lines\n" "" "$(basename "$dir")" "$source_lines" "$test_lines"
    
    # If not a leaf directory, process subdirectories
    if [ "$is_leaf" = "false" ]; then
        for subdir in "$dir"/*/; do
            if [ -d "$subdir" ]; then
                # Check if it's a leaf directory (no .go files in subdirectories)
                if find "$subdir" -mindepth 2 -name "*.go" | grep -q .; then
                    print_stats "$subdir" $((indent + 2)) "false"
                else
                    print_stats "$subdir" $((indent + 2)) "true"
                fi
            fi
        done
    fi
}

# Function to print tree view with line counts
print_tree_with_stats() {
    local dir=$1
    local prefix=$2
    local is_last=$3
    local is_root=$4
    
    # Get line counts
    local source_lines test_lines
    if find "$dir" -mindepth 2 -name "*.go" | grep -q .; then
        read source_lines test_lines < <(count_lines "$dir" 999)
    else
        read source_lines test_lines < <(count_lines "$dir" 1)
    fi
    
    # Print current directory with stats
    if [ "$is_root" = "true" ]; then
        echo "$(basename "$dir") [${source_lines} source, ${test_lines} test]"
    else
        echo "${prefix}└── $(basename "$dir") [${source_lines} source, ${test_lines} test]"
    fi
    
    # Process subdirectories
    local subdirs=("$dir"/*/)
    local count=${#subdirs[@]}
    local i=0
    
    for subdir in "${subdirs[@]}"; do
        if [ -d "$subdir" ]; then
            i=$((i + 1))
            local new_prefix="${prefix}    "
            local is_last_subdir=$([ $i -eq $count ] && echo "true" || echo "false")
            print_tree_with_stats "$subdir" "$new_prefix" "$is_last_subdir" "false"
        fi
    done
}

# Main script
echo "Go Source Code Line Count Statistics"
echo "==================================="
echo

# # Process each target directory
# for dir in cmd pkg service; do
#     if [ -d "$dir" ]; then
#         echo "Directory: $dir"
#         echo "-----------------------------------"
#         print_stats "$dir" 0 "false"
#         echo
#     fi
# done

# # Tree view with line counts
# echo "Tree View with Line Counts"
# echo "======================="
# echo

for dir in cmd pkg service; do
    if [ -d "$dir" ]; then
        print_tree_with_stats "$dir" "" "true" "true"
        echo
    fi
done 