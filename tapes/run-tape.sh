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
  echo "Usage: $0 <file.tape>" >&2
  exit 1
fi

TAPE_FILE="$1"

# Verify the first argument ends with .tape
case "$TAPE_FILE" in
  *.tape) ;;
  *)
    echo "Error: first argument must be a .tape file" >&2
    exit 1
    ;;
esac

# Verify the file exists
if [ ! -f "$TAPE_FILE" ]; then
  echo "Error: file not found: $TAPE_FILE" >&2
  exit 1
fi

# Execute the tape using VHS, replacing the current shell
exec vhs "$TAPE_FILE"
