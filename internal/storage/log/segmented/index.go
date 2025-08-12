package segment

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

const indexEntrySize = 4 + 8 // relativeOffset + filePosition

// relativeOffset is the relative offset of the message
// filePosition is the abosulte byte offset of this message
type indexEntry struct {
	relativeOffset uint32 // 4 bytes
	filePosition   uint64 // 8 bytes
}

type sparseIndex struct {
	f       *os.File
	entries []indexEntry
}

func openSparseIndex(path string) (*sparseIndex, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("segment: error opening index: %w", err)
	}

	ix := &sparseIndex{f: f}
	if err := ix.loadAll(); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("segment: error loading index: %w", err)
	}
	return ix, nil
}

func (ix *sparseIndex) loadAll() error {
	fi, err := ix.f.Stat()
	if err != nil {
		return fmt.Errorf("segment: error fetching file stats: %w", err)
	}
	size := fi.Size()

	if rem := size % indexEntrySize; rem != 0 {
		// truncate index: it might happen for an unsuccessful write during a crash
		err := ix.f.Truncate(size - rem)
		if err != nil {
			return fmt.Errorf("segment: error truncating index file: %w", err)
		}
		size -= size - rem
	}

	if size == 0 {
		ix.entries = nil
		return nil
	}

	buf := make([]byte, size)
	_, err = ix.f.ReadAt(buf, 0)
	if err != nil {
		return fmt.Errorf("segment: error loading index file: %w", err)
	}

	n := size / int64(indexEntrySize)
	entries := make([]indexEntry, 0, n)
	for i := range n {
		start := i * indexEntrySize
		ro := binary.BigEndian.Uint32(buf[start : start+4])
		fp := binary.BigEndian.Uint64(buf[start+4 : start+12])
		entry := indexEntry{
			relativeOffset: ro,
			filePosition:   fp,
		}
		entries = append(entries, entry)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].relativeOffset < entries[j].relativeOffset
	})
	ix.entries = entries
	return nil
}
