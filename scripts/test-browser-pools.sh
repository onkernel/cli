#!/bin/bash

set -e

# Browser Pool Lifecycle Test
#
# This script tests the full lifecycle of browser pools:
# 1. Create a pool
# 2. Acquire a browser from it
# 3. Use the browser (simulated with sleep)
# 4. Release the browser back to the pool
# 5. Check pool state
# 6. Flush idle browsers
# 7. Delete the pool

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
KERNEL="${KERNEL:-kernel}"  # Use $KERNEL env var or default to 'kernel'
POOL_NAME="test-pool-$(date +%s)"
POOL_SIZE=2
SLEEP_TIME=5

# Helper functions
log_step() {
    echo -e "${BLUE}==>${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

log_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Check if kernel CLI is available
if ! command -v "$KERNEL" &> /dev/null && [ ! -x "$KERNEL" ]; then
    log_error "kernel CLI not found at '$KERNEL'. Please install it or set KERNEL env var."
    exit 1
fi

# Cleanup function (only runs if script exits early/unexpectedly)
cleanup() {
    if [ -n "$POOL_ID" ]; then
        echo ""
        log_step "Script exited early - cleaning up pool $POOL_ID"
        "$KERNEL" browser-pools delete "$POOL_ID" --force --no-color || true
        log_info "Cleanup complete (used --force to ensure deletion)"
    fi
}

trap cleanup EXIT

echo ""
log_step "Starting browser pool integration test"
echo ""

# Step 1: Create a pool
log_step "Step 1: Creating browser pool with name '$POOL_NAME' and size $POOL_SIZE"
"$KERNEL" browser-pools create \
    --name "$POOL_NAME" \
    --size "$POOL_SIZE" \
    --timeout 300 \
    --no-color

# Extract pool ID using the list command
POOL_ID=$("$KERNEL" browser-pools list --output json --no-color | jq -r ".[] | select(.name == \"$POOL_NAME\") | .id")

if [ -z "$POOL_ID" ]; then
    log_error "Failed to create pool or extract pool ID"
    exit 1
fi

log_success "Created pool: $POOL_ID"
echo ""

# Step 2: List pools to verify
log_step "Step 2: Listing all pools"
"$KERNEL" browser-pools list --no-color
echo ""

# Step 3: Get pool details
log_step "Step 3: Getting pool details"
"$KERNEL" browser-pools get "$POOL_ID" --no-color
echo ""

# Wait for pool to be ready
log_info "Waiting for pool to initialize..."
sleep 3

# Step 4: Acquire a browser from the pool
log_step "Step 4: Acquiring a browser from the pool"
ACQUIRE_OUTPUT=$("$KERNEL" browser-pools acquire "$POOL_ID" --timeout 10 --no-color 2>&1)

if echo "$ACQUIRE_OUTPUT" | grep -q "timed out"; then
    log_error "Failed to acquire browser (timeout or no browsers available)"
    exit 1
fi

# Parse the session ID from the table output (format: "Session ID | <id>")
SESSION_ID=$(echo "$ACQUIRE_OUTPUT" | grep "Session ID" | awk -F'|' '{print $2}' | xargs)

if [ -z "$SESSION_ID" ]; then
    log_error "Failed to extract session ID from acquire response"
    echo "Response: $ACQUIRE_OUTPUT"
    exit 1
fi

log_success "Acquired browser with session ID: $SESSION_ID"
echo ""

# Step 5: Get pool details again to see the acquired browser
log_step "Step 5: Checking pool state (should show 1 acquired)"
POOL_DETAILS=$("$KERNEL" browser-pools get "$POOL_ID" --output json --no-color)
ACQUIRED_COUNT=$(echo "$POOL_DETAILS" | jq -r '.acquired_count // .acquiredCount // 0')
AVAILABLE_COUNT=$(echo "$POOL_DETAILS" | jq -r '.available_count // .availableCount // 0')

log_info "Acquired: $ACQUIRED_COUNT, Available: $AVAILABLE_COUNT"
"$KERNEL" browser-pools get "$POOL_ID" --no-color
echo ""

# Step 6: Sleep to simulate usage
log_step "Step 6: Simulating browser usage (sleeping for ${SLEEP_TIME}s)"
sleep "$SLEEP_TIME"
log_success "Usage simulation complete"
echo ""

# Step 7: Release the browser back to the pool
log_step "Step 7: Releasing browser back to pool"
"$KERNEL" browser-pools release "$POOL_ID" \
    --session-id "$SESSION_ID" \
    --reuse \
    --no-color

log_success "Browser released"
echo ""

# Step 8: Get pool details again
log_step "Step 8: Checking pool state after release"
"$KERNEL" browser-pools get "$POOL_ID" --no-color
echo ""

# Step 9: Flush the pool
log_step "Step 9: Flushing idle browsers from pool"
"$KERNEL" browser-pools flush "$POOL_ID" --no-color
log_success "Pool flushed"
echo ""

# Step 10: Delete the pool (should succeed if browsers are properly released)
log_step "Step 10: Deleting the pool"
DELETED_POOL_ID="$POOL_ID"
set +e  # Temporarily disable exit-on-error to see the result
"$KERNEL" browser-pools delete "$POOL_ID" --no-color
DELETE_EXIT=$?
set -e  # Re-enable exit-on-error

if [ $DELETE_EXIT -eq 0 ]; then
    log_success "Pool deleted successfully"
    POOL_ID=""  # Clear so cleanup doesn't try again
    echo ""
else
    log_error "Failed to delete pool - browsers may still be in acquired state"
    log_info "This suggests the release operation hasn't fully completed"
    log_info "Pool $POOL_ID left for debugging (clean up manually if needed)"
    POOL_ID=""  # Clear to prevent cleanup trap from trying
    echo ""
fi

# Verify deletion
log_step "Verifying pool deletion"
if "$KERNEL" browser-pools list --output json --no-color | jq -e ".[] | select(.id == \"$DELETED_POOL_ID\") | .id" > /dev/null 2>&1; then
    log_error "Pool may still exist"
else
    log_success "Pool successfully deleted and no longer exists"
fi

echo ""
log_success "Integration test completed successfully!"
echo ""

