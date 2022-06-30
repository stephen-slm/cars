package memory

type Memory int64

const (
	Byte     Memory = 1
	Kilobyte        = 1024 * Byte
	Megabyte        = 1024 * Kilobyte
	Gigabyte        = 1024 * Megabyte
)

func (d Memory) Bytes() int64 { return int64(d) }

func (d Memory) Kilobytes() int64 { return int64(d) / int64(Kilobyte) }

func (d Memory) Megabytes() int64 { return int64(d) / int64(Megabyte) }

func (d Memory) Gigabytes() int64 { return int64(d) / int64(Gigabyte) }

// LimitExceeded is the error returned by the runner if and when the total
// allocated memory has been exceeded.
var LimitExceeded error = memoryLimitExceededError{}

type memoryLimitExceededError struct{}

func (memoryLimitExceededError) Error() string { return "memory limit exceeded" }
