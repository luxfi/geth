#!/bin/bash

# Add skip condition to all long test functions in blockchain_sethead_test.go
file="blockchain_sethead_test.go"

# List of all test functions that need fixing
functions=(
    "testLongShallowSetHead"
    "testLongDeepSetHead"
    "testLongSnapSyncedShallowSetHead"
    "testLongSnapSyncedDeepSetHead"
    "testLongSnapSyncingShallowSetHead"
    "testLongSnapSyncingDeepSetHead"
    "testLongOldForkedShallowSetHead"
    "testLongOldForkedDeepSetHead"
    "testLongOldForkedSnapSyncedShallowSetHead"
    "testLongOldForkedSnapSyncedDeepSetHead"
    "testLongOldForkedSnapSyncingShallowSetHead"
    "testLongOldForkedSnapSyncingDeepSetHead"
    "testLongNewerForkedShallowSetHead"
    "testLongNewerForkedDeepSetHead"
    "testLongNewerForkedSnapSyncedShallowSetHead"
    "testLongNewerForkedSnapSyncedDeepSetHead"
    "testLongNewerForkedSnapSyncingShallowSetHead"
    "testLongNewerForkedSnapSyncingDeepSetHead"
    "testLongReorgedShallowSetHead"
    "testLongReorgedDeepSetHead"
    "testLongReorgedSnapSyncedShallowSetHead"
    "testLongReorgedSnapSyncedDeepSetHead"
    "testLongReorgedSnapSyncingShallowSetHead"
    "testLongReorgedSnapSyncingDeepSetHead"
)

# Add parallel to short tests as well
short_functions=(
    "testShortSetHead"
    "testShortFastSyncedSetHead"
    "testShortFastSyncingSetHead"
    "testShortOldForkedSetHead"
    "testShortOldForkedFastSyncedSetHead"
    "testShortOldForkedFastSyncingSetHead"
    "testShortNewlyForkedSetHead"
    "testShortNewlyForkedFastSyncedSetHead"
    "testShortNewlyForkedFastSyncingSetHead"
    "testShortReorgedSetHead"
    "testShortReorgedFastSyncedSetHead"
    "testShortReorgedFastSyncingSetHead"
    "testShortDeepSetHead"
    "testShortDeepFastSyncedSetHead"
    "testShortDeepFastSyncingSetHead"
)

cp $file ${file}.bak

for func in "${functions[@]}"; do
    # Add skip condition at the beginning of each function
    sed -i "/^func $func(t \*testing.T, snapshots bool) {$/,/^[[:space:]]*\/\/ Chain:$/ {
        s/^func $func(t \*testing.T, snapshots bool) {$/&\n\tif testing.Short() {\n\t\tt.Skip(\"skipping long test in short mode\")\n\t}/
    }" $file
done

for func in "${short_functions[@]}"; do
    # Add parallel to short tests
    sed -i "/^func $func(t \*testing.T, snapshots bool) {$/,/^[[:space:]]*\/\/ Chain:$/ {
        s/^func $func(t \*testing.T, snapshots bool) {$/&\n\tt.Parallel()/
    }" $file
done

echo "Added skip conditions to all long test functions"