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

func (ix *sparseIndex) appendEntry(e indexEntry) error {

	if len(ix.entries) > 0 {
		last := ix.entries[len(ix.entries)-1]
		if e.relativeOffset < last.relativeOffset {
			return fmt.Errorf("segment: non monotonic append: new=%d last=%d", e.relativeOffset, last.relativeOffset)
		}
		if e.relativeOffset == last.relativeOffset {
			return nil
		}
	}

	buf := make([]byte, indexEntrySize)
	binary.BigEndian.PutUint32(buf[0:4], e.relativeOffset)
	binary.BigEndian.PutUint64(buf[4:12], e.filePosition)

	if _, err := ix.f.WriteAt(buf, 0); err != nil {
		return fmt.Errorf("segment: error writing index to file: %w", err)
	}

	ix.entries = append(ix.entries, e)
	return nil
}

func (ix *sparseIndex) searchNearest(target uint32) (indexEntry, bool) {
	if target < ix.entries[0].relativeOffset {
		return indexEntry{}, false
	}

	if len(ix.entries) == 0 {
		return indexEntry{}, false
	}

	// use binary search to find nearest relative offset to target
	lo, hi := 0, len(ix.entries)-1

	for lo <= hi {
		mid := lo + (hi-lo)/2
		ro := ix.entries[mid].relativeOffset
		if ro == target {
			return ix.entries[mid], true
		}
		if ro < target {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	if hi >= 0 {
		return ix.entries[hi], true
	}
	return indexEntry{}, false
}

func (ix *sparseIndex) lastRelativeOffset() (uint32, bool) {
	if len(ix.entries) == 0 {
		return 0, false
	}
	return ix.entries[len(ix.entries)-1].relativeOffset, true
}
