package log

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "00000000.log")

	l, err := Open(Config{
		FilePath:   path,
		RecordSize: 16,
		Fsync:      FsyncAlways,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	p1 := []byte("abcdefghijklmnop")      // 16 bytes
	p2 := []byte(strings.Repeat("x", 16)) // 16 bytes
	p3 := bytes.Repeat([]byte{0x7F}, 16)  // 16 bytes

	off1, err := l.Append(p1)
	if err != nil || off1 != 0 {
		t.Fatalf("Append p1: off=%d err=%v", off1, err)
	}
	off2, err := l.Append(p2)
	if err != nil || off2 != 1 {
		t.Fatalf("Append p2: off=%d err=%v", off2, err)
	}
	off3, err := l.Append(p3)
	if err != nil || off3 != 2 {
		t.Fatalf("Append p3: off=%d err=%v", off3, err)
	}

	got1, err := l.Read(0)
	if err != nil || !bytes.Equal(got1, p1) {
		t.Fatalf("Read(0) mismatch err=%v", err)
	}
	got2, err := l.Read(1)
	if err != nil || !bytes.Equal(got2, p2) {
		t.Fatalf("Read(1) mismatch err=%v", err)
	}
	got3, err := l.Read(2)
	if err != nil || !bytes.Equal(got3, p3) {
		t.Fatalf("Read(2) mismatch err=%v", err)
	}
}

func TestBadPayloadSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "00000000.log")

	l, err := Open(Config{
		FilePath:   path,
		RecordSize: 16,
		Fsync:      FsyncAlways,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	_, err = l.Append([]byte("too-short-15b!!")) // 15 bytes
	if err == nil || err != ErrBadPayloadSize {
		t.Fatalf("expected ErrBadPayloadSize, got %v", err)
	}
}

func TestOffsetOutOfRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "00000000.log")

	l, err := Open(Config{
		FilePath:   path,
		RecordSize: 16,
		Fsync:      FsyncAlways,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	_, err = l.Read(0)
	if err == nil || err != ErrOffsetOutOfRange {
		t.Fatalf("expected ErrOffsetOutOfRange, got %v", err)
	}
}

func TestReopenContinuity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "00000000.log")

	l, err := Open(Config{
		FilePath:   path,
		RecordSize: 16,
		Fsync:      FsyncAlways,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	off0, _ := l.Append([]byte(strings.Repeat("A", 16)))
	off1, _ := l.Append([]byte(strings.Repeat("B", 16)))
	if off0 != 0 || off1 != 1 {
		t.Fatalf("unexpected offsets %d %d", off0, off1)
	}
	_ = l.Close()

	l2, err := Open(Config{
		FilePath:   path,
		RecordSize: 16,
		Fsync:      FsyncAlways,
	})
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer l2.Close()

	off2, err := l2.Append([]byte(strings.Repeat("C", 16)))
	if err != nil {
		t.Fatalf("Append after reopen: %v", err)
	}
	if off2 != 2 {
		t.Fatalf("expected next offset 2, got %d", off2)
	}
}

func TestCRCDetection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "00000000.log")

	l, err := Open(Config{
		FilePath:   path,
		RecordSize: 16,
		Fsync:      FsyncAlways,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	_, err = l.Append([]byte(strings.Repeat("Z", 16)))
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Flip one byte in the payload of record 0.
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	defer f.Close()

	pos := recordPosition(0, headerSize+16) + int64(headerSize) + 3 // inside payload
	_, err = f.WriteAt([]byte{0x00}, pos)
	if err != nil {
		t.Fatalf("corrupt write: %v", err)
	}

	_, err = l.Read(0)
	if err == nil || err != ErrCorruptRecord {
		t.Fatalf("expected ErrCorruptRecord, got %v", err)
	}
}
