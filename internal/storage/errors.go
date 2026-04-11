package storage

import "errors"

var (
	errBloomCorrupt = errors.New("storage: bloom filter corrupt")
	errCodecVersion = errors.New("storage: unknown codec version")
	errCodecTrunc   = errors.New("storage: truncated codec data")
	errWALCorrupt   = errors.New("storage: wal record corrupt")
	errSSTCorrupt   = errors.New("storage: sstable corrupt")
)
