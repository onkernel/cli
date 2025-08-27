#!/usr/bin/env bash
set -e -o pipefail

# Resolve script directory for reliable relative paths
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)

# Prepend ../bin (relative to this script) to PATH with highest priority
BIN_DIR="$SCRIPT_DIR/../bin"
export PATH="$BIN_DIR:$PATH"

# Ensure KERNEL_API_KEY is set
if [ -z "${KERNEL_API_KEY:-}" ]; then
  echo "Error: KERNEL_API_KEY must be set in the environment" >&2
  exit 1
fi

# Validate arguments
if [ $# -lt 1 ]; then
  echo "Usage: $0 <file1.tape> [file2.tape] [file3.tape] ..." >&2
  exit 1
fi

# Store all tape files
TAPE_FILES=()

# Validate all arguments end with .tape and exist
for file in "$@"; do
  case "$file" in
    *.tape) ;;
    *)
      echo "Error: argument must be a .tape file: $file" >&2
      exit 1
      ;;
  esac
  
  if [ ! -f "$file" ]; then
    echo "Error: file not found: $file" >&2
    exit 1
  fi
  
  TAPE_FILES+=("$file")
done

# If only one tape file, run it directly
if [ ${#TAPE_FILES[@]} -eq 1 ]; then
  exec vhs "${TAPE_FILES[0]}"
fi

# Run multiple tape files in series
echo "Running ${#TAPE_FILES[@]} tape files in series..."

# Run each tape file sequentially
for tape_file in "${TAPE_FILES[@]}"; do
  echo "Running: $tape_file"
  vhs "$tape_file"
done

echo "All tape files completed."
