#!/bin/bash
# Script to add testing.Short() checks and t.Parallel() to all repair tests

file="blockchain_repair_test.go"

# List of test functions that need fixing
test_funcs=(
    "testShortNewlyForkedRepair"
    "testShortNewlyForkedSnapSyncedRepair"
    "testShortNewlyForkedSnapSyncingRepair"
    "testShortReorgRepair"
    "testShortReorgSnapSyncedRepair"
    "testShortReorgSnapSyncingRepair"
    "testShortDeepRepair"
    "testShortDeepSnapSyncedRepair"
    "testShortDeepSnapSyncingRepair"
    "testShortDeeperRepair"
    "testShortDeeperSnapSyncedRepair"
    "testShortDeeperSnapSyncingRepair"
    "testShortSethead"
    "testLongShallowRepair"
    "testLongDeepRepair"
    "testLongSnapSyncedShallowRepair"
    "testLongSnapSyncedDeepRepair"
    "testLongSnapSyncingShallowRepair"
    "testLongSnapSyncingDeepRepair"
    "testLongOldForkedShallowRepair"
    "testLongOldForkedDeepRepair"
    "testLongOldForkedSnapSyncedShallowRepair"
    "testLongOldForkedSnapSyncedDeepRepair"
    "testLongOldForkedSnapSyncingShallowRepair"
    "testLongOldForkedSnapSyncingDeepRepair"
    "testLongNewlyForkedShallowRepair"
    "testLongNewlyForkedDeepRepair"
    "testLongNewlyForkedSnapSyncedShallowRepair"
    "testLongNewlyForkedSnapSyncedDeepRepair"
    "testLongNewlyForkedSnapSyncingShallowRepair"
    "testLongNewlyForkedSnapSyncingDeepRepair"
    "testLongReorgRepair"
    "testLongReorgSnapSyncedRepair"
    "testLongReorgSnapSyncingRepair"
    "testMediumDeepRepair"
    "testMediumDeepSnapSyncedRepair"
    "testMediumDeepSnapSyncingRepair"
)

for func in "${test_funcs[@]}"; do
    echo "Fixing $func..."
done