package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func readControlFile(path string) ([]byte, error) {
	if err := validateReportPath(path); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file, not a symlink or special file", path)
	}
	file, err := openEvidenceNoFollow(path)
	if err != nil {
		return nil, fmt.Errorf("open %s without following symlinks: %w", path, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, opened) {
		return nil, fmt.Errorf("%s changed while it was opened", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxReportBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) > maxReportBytes {
		return nil, fmt.Errorf("%s exceeds the %d-byte report limit", path, maxReportBytes)
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(opened, after) ||
		after.Size() != int64(len(data)) || after.ModTime() != opened.ModTime() {
		return nil, fmt.Errorf("%s changed while it was read", path)
	}
	return data, nil
}

func writeAtomic(path string, data []byte, stdout io.Writer) error {
	if path == "-" {
		_, err := stdout.Write(data)
		return err
	}
	if path == "" {
		return fmt.Errorf("output path is required")
	}
	absolute, parent, err := secureOutputDestination(path)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".operations-rehearsal-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Link(temporaryPath, absolute); err != nil {
		return fmt.Errorf("install output without replacement: %w", err)
	}
	if err := syncOutputDirectory(parent); err != nil {
		return err
	}
	if err := os.Remove(temporaryPath); err != nil {
		return fmt.Errorf("remove temporary output link: %w", err)
	}
	if err := syncOutputDirectory(parent); err != nil {
		return err
	}
	return nil
}

func secureOutputDestination(path string) (string, string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("resolve output path: %w", err)
	}
	absolute = filepath.Clean(absolute)
	parent := filepath.Dir(absolute)
	info, err := os.Lstat(parent)
	if err != nil {
		return "", "", fmt.Errorf("inspect output directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", "", fmt.Errorf("output parent must be an existing real directory")
	}
	evaluated, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", "", fmt.Errorf("inspect output parent components: %w", err)
	}
	if filepath.Clean(evaluated) != parent {
		return "", "", fmt.Errorf("output parent contains a symlinked path component")
	}
	if _, err := os.Lstat(absolute); err == nil {
		return "", "", fmt.Errorf("output already exists; replacement is refused")
	} else if !os.IsNotExist(err) {
		return "", "", fmt.Errorf("inspect output path: %w", err)
	}
	return absolute, parent, nil
}

func syncOutputDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open output directory for sync: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync output directory: %w", err)
	}
	return nil
}
