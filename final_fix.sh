#!/bin/bash
# Final fixes for remaining lint issues

echo "Applying final fixes..."

# Fix crypto hasherPool issues
sed -i 's/_ = hasherPool.Put(/hasherPool.Put(/g' crypto/crypto.go

# Fix p2p/enode/nodedb issues
sed -i 's/_ = if/if/g' p2p/enode/nodedb.go

# Fix metrics/influxdb issues
sed -i 's/_ = return/return/g' metrics/influxdb/influxdbv1.go

# Fix ethdb/leveldb issues
sed -i 's/_ = return/return/g' ethdb/leveldb/leveldb.go
sed -i 's/_, _ = /_ = /g' ethdb/leveldb/leveldb.go

# Fix ethdb/pebble issues
sed -i 's/_ = if/if/g' ethdb/pebble/pebble.go

# Fix rpc issues
sed -i 's/_, _ = _, err/_, err/g' rpc/http.go
sed -i 's/_ = return/return/g' rpc/stdio.go

echo "Done!"