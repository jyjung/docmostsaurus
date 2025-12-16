package lock

import (
	"fmt"
	"os"
	"syscall"
)

// DefaultLockFile is the default path for the lock file.
// Note: Uses /tmp directory - ensure write permissions in containerized environments.
const DefaultLockFile = "/tmp/docmostsaurus.lock"

// FileLock provides file-based locking to prevent multiple instances
type FileLock struct {
	path string
	file *os.File
}

// NewFileLock creates a new file lock instance with the default lock file path
func NewFileLock() *FileLock {
	return &FileLock{path: DefaultLockFile}
}

// TryLock attempts to acquire the lock. Returns an error if the lock is already held.
func (l *FileLock) TryLock() error {
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		return fmt.Errorf("another instance is already running (lock file: %s)", l.path)
	}

	l.file = file

	// Write PID to lock file
	file.Truncate(0)
	file.Seek(0, 0)
	file.WriteString(fmt.Sprintf("%d\n", os.Getpid()))

	return nil
}

// Unlock releases the lock
func (l *FileLock) Unlock() error {
	if l.file == nil {
		return nil
	}

	if err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}

	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close lock file: %w", err)
	}

	// Remove lock file
	os.Remove(l.path)
	l.file = nil

	return nil
}
