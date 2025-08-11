package log

import "hash/crc32"

// computeCRC32C return the CRC32C (castagnoli) checksum of data
func computeCRC32C(data []byte) uint32 {
	return crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli))
}
