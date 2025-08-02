// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"encoding/binary"

	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/log"
)

var (
	archiveReferenceKey = []byte("ArchiveReferenceHeight")
)

// WriteArchiveReference stores the archive height reference
func WriteArchiveReference(db ethdb.KeyValueWriter, height uint64) {
	value := encodeBlockNumber(height)
	if err := db.Put(archiveReferenceKey, value); err != nil {
		log.Crit("Failed to store archive reference", "err", err)
	}
}

// ReadArchiveReference reads the archive height reference
func ReadArchiveReference(db ethdb.KeyValueReader) *uint64 {
	data, err := db.Get(archiveReferenceKey)
	if err != nil {
		return nil
	}
	height := binary.BigEndian.Uint64(data)
	return &height
}