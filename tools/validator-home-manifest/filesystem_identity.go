package main

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

const readOnlyFilesystemFlag = 1

var readFilesystemIdentity = filesystemIdentity

func filesystemIdentity(path string) (FilesystemIdentity, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return FilesystemIdentity{}, fmt.Errorf("inspect filesystem root %s: %w", path, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return FilesystemIdentity{}, fmt.Errorf("filesystem root %s must be a real directory", path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return FilesystemIdentity{}, fmt.Errorf("filesystem metadata is unavailable for %s", path)
	}
	var statfs syscall.Statfs_t
	if err := syscall.Statfs(path, &statfs); err != nil {
		return FilesystemIdentity{}, fmt.Errorf("inspect filesystem flags for %s: %w", path, err)
	}
	return FilesystemIdentity{
		CanonicalPath: path,
		DeviceID:      uint64(stat.Dev),
		RootInode:     uint64(stat.Ino),
		ReadOnly:      uint64(statfs.Flags)&readOnlyFilesystemFlag != 0,
	}, nil
}

func requireEmptyDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open destination directory %s: %w", path, err)
	}
	defer directory.Close()
	entries, err := directory.Readdirnames(1)
	if err == nil || len(entries) != 0 {
		return fmt.Errorf("destination directory %s is not empty", path)
	}
	if err != io.EOF {
		return fmt.Errorf("inspect destination directory %s: %w", path, err)
	}
	return nil
}

func fileLinkCount(info os.FileInfo) (uint64, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("file link-count metadata is unavailable")
	}
	return uint64(stat.Nlink), nil
}
