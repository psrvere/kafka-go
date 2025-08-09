package log

import "hash/crc32"

// ComputeCRC32C return the CRC32C (castagnoli) checksum of data
func ComputeCRC32C(data []byte) uint32 {
	return crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli))
}
