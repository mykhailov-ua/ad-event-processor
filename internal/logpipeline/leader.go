package logpipeline

import (
	"fmt"
	"os"
	"syscall"
)

type FileLeaderLock struct {
	path string
	file *os.File
}

func NewFileLeaderLock(path string) *FileLeaderLock {
	return &FileLeaderLock{path: path}
}

func (fl *FileLeaderLock) TryAcquire() (bool, error) {
	if fl.file != nil {
		return true, nil
	}

	file, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return false, err
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if err == syscall.EWOULDBLOCK {
			leaderHeld.Set(0)
			return false, nil
		}
		return false, fmt.Errorf("flock %s: %w", fl.path, err)
	}

	fl.file = file
	leaderHeld.Set(1)
	return true, nil
}

func (fl *FileLeaderLock) Release() error {
	if fl.file == nil {
		leaderHeld.Set(0)
		return nil
	}
	err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN)
	closeErr := fl.file.Close()
	fl.file = nil
	leaderHeld.Set(0)
	if err != nil {
		return err
	}
	return closeErr
}

func (fl *FileLeaderLock) Path() string {
	return fl.path
}
