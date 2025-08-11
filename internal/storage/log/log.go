package log

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type FsyncPolicy int

const (
	FsyncAlways FsyncPolicy = iota
	FsyncEveryN
	FsyncInterval
)

var (
	ErrBadPayloadSize   = errors.New("log: payload size mismatch")
	ErrOffsetOutOfRange = errors.New("log: offset out of range")
	ErrCorruptRecord    = errors.New("log: corrupt record")
	ErrClosed           = errors.New("log: closed")

	ErrFilePathRequired = errors.New("log: FilePath is required")
	ErrRecordSizeZero   = errors.New("log: Record size must be > 0")
)

type Config struct {
	FilePath      string
	RecordSize    int
	Fsync         FsyncPolicy
	FsyncEveryN   int
	FsyncInterval time.Duration
	Preallocate   bool
}

type Log struct {
	mu               *sync.RWMutex
	f                *os.File
	recordSize       int
	fileRecordSize   int
	nextOffset       uint64
	fsyncPolicy      FsyncPolicy
	fsyncEveryN      int
	fsyncInterval    time.Duration
	pendingSinceSync int
	closed           bool
}

// Open opens a new/existing file and truncates it to the last
// successful record
func Open(cfg Config) (*Log, error) {
	if cfg.FilePath == "" {
		return nil, ErrFilePathRequired
	}
	if cfg.RecordSize <= 0 {
		return nil, ErrRecordSizeZero
	}

	f, err := os.OpenFile(cfg.FilePath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("log: error opening file: %v", err)
	}

	fileRecordSize := headerSize + cfg.RecordSize

	fstat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("log: error getting file stats: %v", err)
	}

	size := fstat.Size()
	// Trim down to last full record i.e. discard any partial writes
	// May last write crashed
	if rem := size % int64(fileRecordSize); rem != 0 {
		newSize := size - rem
		if err := f.Truncate(newSize); err != nil {
			f.Close()
			return nil, fmt.Errorf("log: error truncating file: %v", err)
		}
	}

	next := uint64(size / int64(fileRecordSize))

	l := &Log{
		mu:               &sync.RWMutex{},
		f:                f,
		recordSize:       cfg.RecordSize,
		fileRecordSize:   fileRecordSize,
		nextOffset:       next,
		fsyncPolicy:      cfg.Fsync,
		fsyncEveryN:      cfg.FsyncEveryN,
		fsyncInterval:    cfg.FsyncInterval,
		pendingSinceSync: 0,
	}
	return l, nil
}

// Append appends the new record
// Returns offset and error
// Offset is returned for caller so that they don't have to make another call
// to get the offset
func (l *Log) Append(payload []byte) (uint64, error) {
	if len(payload) != l.recordSize {
		return 0, ErrBadPayloadSize
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return 0, ErrClosed
	}

	offset := l.nextOffset
	pos := recordPosition(offset, l.fileRecordSize)
	record := getCRCRecord(payload, l.fileRecordSize)

	if _, err := l.f.WriteAt(record, pos); err != nil {
		return 0, fmt.Errorf("log: error appending to file: %v", err)
	}

	l.nextOffset++
	l.pendingSinceSync++

	switch l.fsyncPolicy {
	case FsyncAlways:
		if err := l.f.Sync(); err != nil {
			return 0, fmt.Errorf("log: error saving data to disk: %v", err)
		}
		l.pendingSinceSync = 0
		// TODO: Implement FsyncEveryN and FsyncInterval
	}

	return offset, nil
}

func (l *Log) Read(offset uint64) ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.closed {
		return nil, ErrClosed
	}

	if offset >= l.nextOffset {
		return nil, ErrOffsetOutOfRange
	}

	pos := recordPosition(offset, l.fileRecordSize)
	buf := make([]byte, l.fileRecordSize)
	if _, err := l.f.ReadAt(buf, pos); err != nil {
		return nil, fmt.Errorf("log: Error reading file: %v", err)
	}

	if !verifyCRC(buf) {
		return nil, ErrCorruptRecord
	}

	payload := buf[headerSize:]
	out := make([]byte, len(payload))
	copy(out, payload)
	return out, nil

	// Here we copy payload value into a new array to decouple client's copy
	// of array wiht our service. This allows us to enable buffer pooling or using
	// scratch buffer across calls in future
}

func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true
	// Sync remaining data before closing

	if err := l.f.Sync(); err != nil {
		_ = l.f.Close()
		return fmt.Errorf("log: error saving data to disk: %v", err)
	}
	l.pendingSinceSync = 0

	if err := l.f.Close(); err != nil {
		return fmt.Errorf("log: error closing file: %v", err)
	}

	return nil
}
