package memory

type MemorySize int64

const (
	Byte     MemorySize = 1
	Kilobyte            = 1024 * Byte
	Megabyte            = 1024 * Kilobyte
	Gigabyte            = 1024 * Megabyte
)

func (d MemorySize) Bytes() int64 { return int64(d) }

func (d MemorySize) Kilobytes() int64 { return int64(d) / int64(Kilobyte) }

func (d MemorySize) Megabytes() int64 { return int64(d) / int64(Megabyte) }

func (d MemorySize) Gigabytes() int64 { return int64(d) / int64(Gigabyte) }
