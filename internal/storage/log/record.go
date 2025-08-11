package log

import "encoding/binary"

const headerSize = 4

func recordPosition(offset uint64, fileRecordSize int) int64 {
	return int64(offset) * int64(fileRecordSize)
}

func getCRCRecord(payload []byte, fileRecordSize int) []byte {
	crc := computeCRC32C(payload)
	// create a buffer of fileRecordSize instead of headerSize to avoid reallocation
	// when header and record bytes are combined
	buf := make([]byte, fileRecordSize)
	// byte(value) will truncate crc to 8 bits as byte is unit8 in Go
	binary.BigEndian.PutUint32(buf[:headerSize], crc)

	// copy is more efficient than append(buf, payload...)
	copy(buf[headerSize:], payload)
	return buf
}

func verifyCRC(record []byte) bool {
	haveCRC := record[:headerSize]
	payload := record[headerSize:]
	gotCRC := computeCRC32C(payload)
	if binary.BigEndian.Uint32(haveCRC) == gotCRC {
		return true
	}
	return false
}
