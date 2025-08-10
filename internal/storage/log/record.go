package log

import "encoding/binary"

const headerSize = 4

func recordPosition(offset uint64, fileRecordSize int) int64 {
	return int64(offset) * int64(fileRecordSize)
}

func getCRCRecord(payload []byte) []byte {
	crc := ComputeCRC32C(payload)
	buf := make([]byte, headerSize)
	binary.BigEndian.PutUint32(buf, crc) // byte(value) will truncate crc to 8 bits as byte is unit8 in Go

	return append(buf, payload...)
}
