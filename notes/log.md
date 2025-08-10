These are the notes I made while learning and implementing the Kafka log data structure in this project

## What is CRC

- CRC (Cyclic Redundancy Check): a fast checksum to detect data corruption
- CRC32C: a 32-bit CRC using Castagnoli polynomial; widely used in storage (Kafka, LevelDB, SCTP)

It is commonly used in digital networks, storage devices and data transmission protocols to identify accidental changes or corruption in data. It takes data as a binary number, appends a fixed-length checksum (the CRC value). When the data is received or read, the CRC is computed and compared to the appended value.

CRC comes in various bit lengths (e.g. 8-bit, 16-bit, 32-bit) and use different generator polynomials which define their error detection properties.

Common Variants: 
  - CRC32 (CRC-32-IEEE): originally standardized for Ethernet
  - CRC32C (Castagnoli CRC): used in storage and high speed networks

## Log File Design

Kafka logs are append-only sequences of messages (records). In our design the log file is a flat, contiguous stream of fixed-size records, each prefixed with a checksum for integrity.

- Flat: simple, linear, non-hierarchical structure - no nested sections, metadata blocks, or complex organization. It is just a straightforward sequence of records. Alternatives are hierarchical (JSON, XML), segmented (separate header/footer/etc), or database-like (B-trees).
- Contiguous Stream: records are packed back-to-back without gaps or padding between them. One record ends where the next begins, forming a continuous data flow. Alternatives are non-contiguous (pointer-based records).

There is no variable-length complexity or heavy metadata. The core idea is to treat the file as an array of fixed-size slots, where each slot holds each record. This allows accessing data fast without needing an index file - O(1) time complexity for searching records.

### File Layout

Each Record consists of:
- CRC32C Header: 4-byte checksum stored as a 32-bit integer
- Payload: This is exactly `RecordSize` bytes. This is the actual Kafka message data. For our design, payload must be exactly this size because we do not support padding logic as of now. For now, the caller has to handle it.

`FileRecordSize` = `RecordSize` (payload) + 4 (CRC)

Each Record is fixed-size. Why? Variable-length records add two complexities
- storing length per record
- maintaining indexing for fast access

### Offsets and Positioning

Offsets identify the location of a record, starting from 0
- First record: at 0
- Second record: at `FileRecordSize`
- Nth record: at `N * FileRecordSize`

For appending a new record, `offset = filesize/FileRecordSize`

This design's simplicity keeps lookup time complexity at O(1)

### Log File Operations

- Appending a new Record: Find offset by `filesize/FileRecordSize` and append new record at this position.
- Reading a new Record: Calculate offset for N as `N * FileRecordSize` and read `FileRecordSize` bytes from this location. The first 4 bytes are checksum; the rest is data. CRC is verified before returning data.
- Handling Errors:
  - Crash scenario 1: append operation just wrote 2 bytes of data of, say, 20 bytes of total data. In our design, we overwrite these 2 bytes as our offset is a multiple of 20 bytes.
  - CRC verification failed: stop at first failed CRC. Records after that will be overwritten (discarded)

### CRC32C Big Endianness Property

Endianness is how a computer architecture stores and transmits byte data
Big Endianness: stores the most significant byte first. E.g., 0x12345678 stored as [0x12, 0x34, 0x56, 0x78]
Little Endianness: stores the least significant byte first. E.g., 0x12345678 stored as [0x78, 0x56, 0x34, 0x12]

Since the CRC32C value is used as a checksum, it needs to be consistent across computer architectures.