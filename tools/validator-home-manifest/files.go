package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

var requiredDatabaseRoots = []struct {
	name string
	root string
}{
	{name: "application", root: "data/application.db"},
	{name: "blockstore", root: "data/blockstore.db"},
	{name: "comet_state", root: "data/state.db"},
}

func secureAbsoluteDirectory(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", path, err)
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect %s: %w", absolute, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%s is a symlink", absolute)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", absolute)
	}
	evaluated, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks in %s: %w", absolute, err)
	}
	// System-level aliases such as macOS /var -> /private/var are resolved to
	// a stable canonical root. The home path itself and every entry beneath it
	// are still inspected with Lstat and may not be symlinks.
	return filepath.Clean(evaluated), nil
}

func readRegularFile(path string, limit int64) ([]byte, os.FileInfo, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect %s: %w", path, err)
	}
	if !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s must be a regular file", path)
	}
	file, err := openRegularNoFollow(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("stat open %s: %w", path, err)
	}
	if !os.SameFile(before, opened) || !opened.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s changed while it was opened", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	if int64(len(data)) > limit {
		return nil, nil, fmt.Errorf("%s exceeds the %d-byte limit", path, limit)
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, after) || after.Size() != before.Size() ||
		after.ModTime() != before.ModTime() || after.Mode() != before.Mode() {
		return nil, nil, fmt.Errorf("%s changed while it was read", path)
	}
	return data, opened, nil
}

func hashRegularFile(path string) (FileRecord, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return FileRecord{}, fmt.Errorf("inspect %s: %w", path, err)
	}
	if !before.Mode().IsRegular() {
		return FileRecord{}, fmt.Errorf("%s is not a regular file", path)
	}
	file, err := openRegularNoFollow(path)
	if err != nil {
		return FileRecord{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return FileRecord{}, fmt.Errorf("stat %s: %w", path, err)
	}
	if !os.SameFile(before, opened) || !opened.Mode().IsRegular() {
		return FileRecord{}, fmt.Errorf("%s changed while it was opened", path)
	}
	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return FileRecord{}, fmt.Errorf("hash %s: %w", path, err)
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, after) || after.Size() != size ||
		after.ModTime() != before.ModTime() || after.Mode() != before.Mode() {
		return FileRecord{}, fmt.Errorf("%s changed while it was hashed", path)
	}
	return FileRecord{
		Size:   size,
		Mode:   modeText(opened.Mode()),
		SHA256: hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func openRegularNoFollow(path string) (*os.File, error) {
	fd, err := syscall.Open(
		path,
		syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("open %s without following symlinks: %w", path, err)
	}
	return os.NewFile(uintptr(fd), path), nil
}

func modeText(mode fs.FileMode) string {
	return fmt.Sprintf("%04o", mode.Perm())
}

func scanHome(home string) (ContentManifest, error) {
	content := ContentManifest{
		Directories: make([]DirectoryRecord, 0),
		Files:       make([]FileRecord, 0),
	}
	err := filepath.WalkDir(home, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %s: %w", path, walkErr)
		}
		if path == home {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect %s: %w", path, err)
		}
		relative, err := filepath.Rel(home, path)
		if err != nil {
			return fmt.Errorf("make %s relative to home: %w", path, err)
		}
		relative = filepath.ToSlash(relative)
		if relative == "." || relative == "" || strings.HasPrefix(relative, "../") {
			return fmt.Errorf("unsafe relative path %q", relative)
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("refuse symlink %s", relative)
		case info.IsDir():
			content.Directories = append(content.Directories, DirectoryRecord{
				Path: relative,
				Mode: modeText(info.Mode()),
			})
		case info.Mode().IsRegular():
			record, err := hashRegularFile(path)
			if err != nil {
				return err
			}
			record.Path = relative
			content.Files = append(content.Files, record)
		default:
			return fmt.Errorf("refuse special file %s with mode %s", relative, info.Mode())
		}
		return nil
	})
	if err != nil {
		return ContentManifest{}, err
	}
	sort.Slice(content.Directories, func(i, j int) bool {
		return content.Directories[i].Path < content.Directories[j].Path
	})
	sort.Slice(content.Files, func(i, j int) bool {
		return content.Files[i].Path < content.Files[j].Path
	})
	contentForHash := content
	contentForHash.SHA256 = ""
	digest, err := hashCanonical(contentForHash)
	if err != nil {
		return ContentManifest{}, err
	}
	content.SHA256 = digest
	return content, nil
}

func databaseManifests(content ContentManifest) ([]DatabaseManifest, error) {
	databases := make([]DatabaseManifest, 0, len(requiredDatabaseRoots))
	directories := make(map[string]struct{}, len(content.Directories))
	for _, directory := range content.Directories {
		directories[directory.Path] = struct{}{}
	}
	for _, required := range requiredDatabaseRoots {
		if _, exists := directories[required.root]; !exists {
			return nil, fmt.Errorf("required %s database directory %s is missing", required.name, required.root)
		}
		database := DatabaseManifest{
			Name:  required.name,
			Root:  required.root,
			Files: make([]FileRecord, 0),
		}
		prefix := required.root + "/"
		for _, file := range content.Files {
			if strings.HasPrefix(file.Path, prefix) {
				database.Files = append(database.Files, file)
			}
		}
		if len(database.Files) == 0 {
			return nil, fmt.Errorf("required %s database %s contains no regular files", required.name, required.root)
		}
		databaseForHash := database
		databaseForHash.SHA256 = ""
		digest, err := hashCanonical(databaseForHash)
		if err != nil {
			return nil, err
		}
		database.SHA256 = digest
		databases = append(databases, database)
	}
	return databases, nil
}

func ensureOutsideHome(home, output string) error {
	if output == "" || output == "-" {
		return nil
	}
	absolute, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	relative, err := filepath.Rel(home, filepath.Clean(absolute))
	if err != nil {
		return fmt.Errorf("compare output path to home: %w", err)
	}
	if relative == "." || (!strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != "..") {
		return fmt.Errorf("output path must be outside the validator home")
	}
	return nil
}

func writeAtomic(path string, data []byte, output io.Writer) error {
	if path == "-" {
		_, err := output.Write(data)
		return err
	}
	if path == "" {
		return fmt.Errorf("output path is required")
	}
	absolute, parent, err := secureOutputDestination(path)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".validator-home-manifest-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure temporary output: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary output: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}
	if err := os.Link(temporaryPath, absolute); err != nil {
		return fmt.Errorf("install output without replacement: %w", err)
	}
	if err := syncDirectory(parent); err != nil {
		return err
	}
	if err := os.Remove(temporaryPath); err != nil {
		return fmt.Errorf("remove temporary output link: %w", err)
	}
	if err := syncDirectory(parent); err != nil {
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

func syncDirectory(path string) error {
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
