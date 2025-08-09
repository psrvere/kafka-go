These are the notes I made while implementing log in this project

## What is CRC

- CRC (Cyclic Redundancy Check): a fast checksum to detect data corruption
- CRC32C: a 32-bit CRC using Castagnoli polynomial; widely used in storage (Kafka, LevelDB, SCTP)

It is commonly used in digital networks, storage devices and data transmission protocols to identify accidental changes or corruption in data. It takes data as a binary number, appends a fixed-length checksum (the CRC value). When the data is received or read, the CRC is computed and compared to the appended value.

CRC comes in various bit lengths (e.g. 8-bit, 16-bit, 32-bit) and use different generator polynomials which define their error detection properties.

Common Variants: 
  - CRC32 (CRC-32-IEEE): originally standardized for Ethernet
  - CRC32C (Castagnoli CRC): used in storage and high speed networks