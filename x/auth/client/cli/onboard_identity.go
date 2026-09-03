package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/zerone-chain/zerone/x/auth/types"
)

const (
	darwinACLHelperName       = "darwin-acl-check"
	darwinACLClearResult      = "zerone-darwin-acl-v1 clear\n"
	darwinACLPresentResult    = "zerone-darwin-acl-v1 present\n"
	darwinACLPresentExit      = 10
	darwinACLHelperMode       = 0o555
	maxDarwinACLHelperBytes   = 1 << 20
	maxDarwinACLProtocolBytes = 4 << 10
	darwinACLInspectionLimit  = 10 * time.Second
	darwinACLWaitDelay        = 500 * time.Millisecond
)

var (
	darwinACLHelperPathOverride string // Tests only; production resolves beside zeroned.
	fatMachOMagics              = map[[4]byte]struct{}{
		{0xca, 0xfe, 0xba, 0xbe}: {},
		{0xca, 0xfe, 0xba, 0xbf}: {},
	}
)

type onboardIdentityLocation struct {
	directory     *os.File
	directoryPath string
	name          string
	path          string
}

// secureOnboardIdentityLocation walks the absolute parent path exclusively via
// directory descriptors. Missing components are created one at a time, then
// owner/mode/ACL checked and durably linked before the next edge is traversed.
func secureOnboardIdentityLocation(path string, createParent bool) (*onboardIdentityLocation, error) {
	if path == "" || strings.HasSuffix(path, string(filepath.Separator)) {
		return nil, fmt.Errorf("identity file path must name a file")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve identity file path: %w", err)
	}
	absPath = filepath.Clean(absPath)
	name := filepath.Base(absPath)
	if name == "" || name == "." || name == ".." {
		return nil, fmt.Errorf("identity file path must have a safe basename")
	}
	parentPath, err := canonicalIdentityParent(filepath.Dir(absPath))
	if err != nil {
		return nil, err
	}

	rootDescriptor, err := unix.Open(
		string(filepath.Separator),
		unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("open filesystem root for identity custody: %w", err)
	}
	current := os.NewFile(uintptr(rootDescriptor), string(filepath.Separator))
	if current == nil {
		_ = unix.Close(rootDescriptor)
		return nil, fmt.Errorf("bind filesystem root descriptor for identity custody")
	}

	components := strings.Split(strings.TrimPrefix(parentPath, string(filepath.Separator)), string(filepath.Separator))
	if len(components) == 1 && components[0] == "" {
		components = nil
	}
	currentPath := string(filepath.Separator)
	for _, component := range components {
		if component == "" || component == "." || component == ".." {
			_ = current.Close()
			return nil, fmt.Errorf("identity directory contains an unsafe component")
		}
		nextPath := filepath.Join(currentPath, component)
		childDescriptor, openErr := unix.Openat(
			int(current.Fd()),
			component,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK,
			0,
		)
		created := false
		appearedDuringCreate := false
		if openErr != nil && errors.Is(openErr, unix.ENOENT) && createParent {
			if mkdirErr := unix.Mkdirat(int(current.Fd()), component, 0o700); mkdirErr != nil {
				if !errors.Is(mkdirErr, unix.EEXIST) {
					_ = current.Close()
					return nil, fmt.Errorf("create private identity directory %s: %w", nextPath, mkdirErr)
				}
				appearedDuringCreate = true
			} else {
				created = true
			}
			childDescriptor, openErr = unix.Openat(
				int(current.Fd()),
				component,
				unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK,
				0,
			)
		}
		if openErr != nil {
			_ = current.Close()
			if errors.Is(openErr, unix.ENOENT) {
				return nil, &os.PathError{Op: "open identity directory", Path: nextPath, Err: openErr}
			}
			return nil, fmt.Errorf("secure no-follow open of identity directory %s: %w", nextPath, openErr)
		}
		child := os.NewFile(uintptr(childDescriptor), nextPath)
		if child == nil {
			_ = unix.Close(childDescriptor)
			_ = current.Close()
			return nil, fmt.Errorf("bind identity directory descriptor %s", nextPath)
		}

		if created {
			if err := child.Chmod(0o700); err != nil {
				_ = child.Close()
				_ = current.Close()
				return nil, fmt.Errorf("protect new identity directory %s: %w", nextPath, err)
			}
		}
		if created || appearedDuringCreate {
			if err := validatePrivateOnboardDirectory(child, nextPath, true); err != nil {
				_ = child.Close()
				_ = current.Close()
				return nil, err
			}
			// Persist the new inode and then the name that reaches it before
			// advancing. Key generation cannot occur until the full chain passes.
			if err := child.Sync(); err != nil {
				_ = child.Close()
				_ = current.Close()
				return nil, fmt.Errorf("sync identity directory %s: %w", nextPath, err)
			}
			if err := current.Sync(); err != nil {
				_ = child.Close()
				_ = current.Close()
				return nil, fmt.Errorf("sync containing directory for %s: %w", nextPath, err)
			}
		}

		if err := current.Close(); err != nil {
			_ = child.Close()
			return nil, fmt.Errorf("close traversed identity directory %s: %w", currentPath, err)
		}
		current = child
		currentPath = nextPath
	}

	if err := validatePrivateOnboardDirectory(current, parentPath, false); err != nil {
		_ = current.Close()
		return nil, err
	}
	return &onboardIdentityLocation{
		directory:     current,
		directoryPath: parentPath,
		name:          name,
		path:          absPath,
	}, nil
}

// canonicalIdentityParent permits platform-owned aliases such as macOS /var ->
// /private/var without ever traversing them during custody. Only the deepest
// existing ancestor is resolved; every resulting component is then opened
// with O_NOFOLLOW and every missing component is created relative to a held fd.
func canonicalIdentityParent(requested string) (string, error) {
	probe := requested
	missing := make([]string, 0)
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			for _, component := range missing {
				resolved = filepath.Join(resolved, component)
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("resolve existing identity directory ancestor %s: %w", probe, err)
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", fmt.Errorf("identity directory %s has no existing ancestor", requested)
		}
		missing = append([]string{filepath.Base(probe)}, missing...)
		probe = parent
	}
}

func (location *onboardIdentityLocation) close() error {
	if location == nil || location.directory == nil {
		return nil
	}
	err := location.directory.Close()
	location.directory = nil
	return err
}

// assertReachable re-walks the canonical path from / with O_NOFOLLOW and
// verifies that it still reaches the held final-parent inode. This detects an
// ancestor rename that would otherwise leave a safe descriptor pointing at an
// orphaned custody directory while a replacement path is shown to the user.
func (location *onboardIdentityLocation) assertReachable() error {
	if location == nil || location.directory == nil {
		return fmt.Errorf("identity custody directory is closed")
	}
	if err := validatePrivateOnboardDirectory(location.directory, location.directoryPath, false); err != nil {
		return err
	}
	heldInfo, err := location.directory.Stat()
	if err != nil {
		return fmt.Errorf("inspect held identity directory %s: %w", location.directoryPath, err)
	}
	reopened, err := openCanonicalDirectoryNoFollow(location.directoryPath)
	if err != nil {
		return fmt.Errorf("identity directory %s is no longer reachable through its canonical path: %w", location.directoryPath, err)
	}
	defer reopened.Close()
	if err := validatePrivateOnboardDirectory(reopened, location.directoryPath, false); err != nil {
		return err
	}
	reopenedInfo, err := reopened.Stat()
	if err != nil {
		return fmt.Errorf("inspect re-opened identity directory %s: %w", location.directoryPath, err)
	}
	if !os.SameFile(heldInfo, reopenedInfo) {
		return fmt.Errorf("identity directory %s changed or was orphaned during custody operation", location.directoryPath)
	}
	return nil
}

func openCanonicalDirectoryNoFollow(path string) (*os.File, error) {
	rootDescriptor, err := unix.Open(
		string(filepath.Separator),
		unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK,
		0,
	)
	if err != nil {
		return nil, err
	}
	current := os.NewFile(uintptr(rootDescriptor), string(filepath.Separator))
	if current == nil {
		_ = unix.Close(rootDescriptor)
		return nil, fmt.Errorf("bind filesystem root descriptor")
	}
	components := strings.Split(strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator)), string(filepath.Separator))
	if len(components) == 1 && components[0] == "" {
		components = nil
	}
	for _, component := range components {
		if component == "" || component == "." || component == ".." {
			_ = current.Close()
			return nil, fmt.Errorf("unsafe canonical path component")
		}
		descriptor, openErr := unix.Openat(
			int(current.Fd()),
			component,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK,
			0,
		)
		if openErr != nil {
			_ = current.Close()
			return nil, openErr
		}
		next := os.NewFile(uintptr(descriptor), filepath.Join(current.Name(), component))
		if next == nil {
			_ = unix.Close(descriptor)
			_ = current.Close()
			return nil, fmt.Errorf("bind canonical directory descriptor")
		}
		if err := current.Close(); err != nil {
			_ = next.Close()
			return nil, err
		}
		current = next
	}
	return current, nil
}

func validatePrivateOnboardDirectory(directory *os.File, path string, requireExactMode bool) error {
	info, err := directory.Stat()
	if err != nil {
		return fmt.Errorf("inspect identity directory %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("identity directory %s is not a directory", path)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("identity directory %s permissions must not allow group or world access", path)
	}
	if requireExactMode && info.Mode().Perm() != 0o700 {
		return fmt.Errorf("new identity directory %s must have exact mode 0700", path)
	}
	var status unix.Stat_t
	if err := unix.Fstat(int(directory.Fd()), &status); err != nil {
		return fmt.Errorf("inspect identity directory owner %s: %w", path, err)
	}
	if status.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("identity directory %s must be owned by the current user", path)
	}
	if err := validateNoDarwinExtendedACL(directory, path, "directory"); err != nil {
		return err
	}
	return nil
}

func readOnboardIdentity(path, expectedAddress string) (onboardIdentity, error) {
	location, err := secureOnboardIdentityLocation(path, false)
	if err != nil {
		return onboardIdentity{}, err
	}
	defer location.close()
	return readOnboardIdentityAt(location, expectedAddress)
}

func readOnboardIdentityAt(location *onboardIdentityLocation, expectedAddress string) (onboardIdentity, error) {
	var identity onboardIdentity
	descriptor, err := unix.Openat(
		int(location.directory.Fd()),
		location.name,
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK,
		0,
	)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return identity, &os.PathError{Op: "open", Path: location.path, Err: err}
		}
		return identity, fmt.Errorf("identity file must be a regular non-symlink file (secure open failed: %w)", err)
	}
	file := os.NewFile(uintptr(descriptor), location.path)
	if file == nil {
		_ = unix.Close(descriptor)
		return identity, fmt.Errorf("open identity file descriptor")
	}
	defer file.Close()

	if err := validateOnboardIdentityFile(file, location.path); err != nil {
		return identity, err
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxOnboardIdentityBytes+1))
	if err != nil {
		return identity, fmt.Errorf("read identity file: %w", err)
	}
	if len(raw) > maxOnboardIdentityBytes {
		return identity, fmt.Errorf("identity file exceeds %d-byte limit", maxOnboardIdentityBytes)
	}
	if err := validateOnboardIdentityFile(file, location.path); err != nil {
		return identity, err
	}
	if err := json.Unmarshal(raw, &identity); err != nil {
		return identity, fmt.Errorf("decode identity file: %w", err)
	}
	if identity.Address != expectedAddress {
		return identity, fmt.Errorf("identity belongs to %q, not %q", identity.Address, expectedAddress)
	}
	seed, err := hex.DecodeString(identity.PrivateKeyHex)
	if err != nil || len(seed) != ed25519.SeedSize {
		return identity, fmt.Errorf("identity file has a malformed private key")
	}
	derived := hex.EncodeToString(ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey))
	if derived != identity.PublicKeyHex || identity.Did != "did:zrn:"+derived {
		return identity, fmt.Errorf("identity file is inconsistent (public key does not derive from private key)")
	}
	if _, err := types.DecodeEd25519PublicKeyHex(identity.PublicKeyHex); err != nil {
		return identity, fmt.Errorf("identity file has an invalid public key: %w", err)
	}
	return identity, nil
}

func validateOnboardIdentityFile(file *os.File, path string) error {
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect opened identity file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("identity file must be a regular non-symlink file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("identity file permissions must not allow group or world access")
	}
	if info.Size() < 0 || info.Size() > maxOnboardIdentityBytes {
		return fmt.Errorf("identity file exceeds %d-byte limit", maxOnboardIdentityBytes)
	}
	var status unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &status); err != nil {
		return fmt.Errorf("inspect identity file owner: %w", err)
	}
	if status.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("identity file must be owned by the current user")
	}
	return validateNoDarwinExtendedACL(file, path, "file")
}

func persistOnboardIdentity(path string, identity onboardIdentity) error {
	location, err := secureOnboardIdentityLocation(path, true)
	if err != nil {
		return err
	}
	defer location.close()
	return persistOnboardIdentityAt(location, identity)
}

func persistOnboardIdentityAt(location *onboardIdentityLocation, identity onboardIdentity) error {
	if err := validatePrivateOnboardDirectory(location.directory, location.directoryPath, false); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return fmt.Errorf("encode identity file: %w", err)
	}
	if len(raw) > maxOnboardIdentityBytes {
		return fmt.Errorf("identity file exceeds %d-byte limit", maxOnboardIdentityBytes)
	}
	descriptor, err := unix.Openat(
		int(location.directory.Fd()),
		location.name,
		unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0o600,
	)
	if err != nil {
		return &os.PathError{Op: "create", Path: location.path, Err: err}
	}
	file := os.NewFile(uintptr(descriptor), location.path)
	if file == nil {
		_ = unix.Close(descriptor)
		return fmt.Errorf("open new identity file descriptor")
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()

	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("protect identity file permissions: %w", err)
	}
	// Inspect before writing: an inherited Darwin ACL must never receive a
	// single byte of the seed, even transiently.
	if err := validateOnboardIdentityFile(file, location.path); err != nil {
		return err
	}
	if err := writeAllOnboardIdentity(file, raw); err != nil {
		return fmt.Errorf("write identity file: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync identity file: %w", err)
	}
	if err := validateOnboardIdentityFile(file, location.path); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close identity file: %w", err)
	}
	closed = true
	if err := location.directory.Sync(); err != nil {
		return fmt.Errorf("sync identity directory: %w", err)
	}
	return location.assertReachable()
}

func writeAllOnboardIdentity(file *os.File, raw []byte) error {
	for len(raw) > 0 {
		written, err := file.Write(raw)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		raw = raw[written:]
	}
	return nil
}

// validateNoDarwinExtendedACL delegates only the already-open target as fd 0.
// Mode checks cannot reveal macOS NFSv4 ACL grants. Linux chmod constrains the
// effective POSIX ACL mask, while unrecognized platforms fail closed.
func validateNoDarwinExtendedACL(target *os.File, path, kind string) error {
	if runtime.GOOS == "linux" {
		return nil
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("refusing identity %s %s: ACL policy is unsupported on %s", kind, path, runtime.GOOS)
	}

	helperPath, err := resolveDarwinACLHelperPath()
	if err != nil {
		return fmt.Errorf("refusing identity %s %s: descriptor ACL inspection is unavailable or unsafe: %w", kind, path, err)
	}
	helperDirectoryPath := filepath.Dir(helperPath)
	helperDirectory, helperDirectoryInfo, err := openValidatedDarwinACLHelperDirectory(helperDirectoryPath)
	if err != nil {
		return fmt.Errorf("refusing identity %s %s: descriptor ACL inspection is unavailable or unsafe: %w", kind, path, err)
	}
	defer helperDirectory.Close()

	helper, helperInfo, err := openValidatedDarwinACLHelper(helperPath)
	if err != nil {
		return fmt.Errorf("refusing identity %s %s: descriptor ACL inspection is unavailable or unsafe: %w", kind, path, err)
	}
	defer helper.Close()
	if err := assertDarwinACLHelperDirectoryUnchanged(helperDirectoryPath, helperDirectoryInfo); err != nil {
		return fmt.Errorf("refusing identity %s %s: descriptor ACL inspection is unavailable or unsafe: %w", kind, path, err)
	}
	// One aggregate deadline covers helper bootstrap and target inspection. It
	// tolerates process-start latency under system load without turning three
	// individual calls into a 30-second failure path.
	ctx, cancel := context.WithTimeout(context.Background(), darwinACLInspectionLimit)
	defer cancel()

	for _, bootstrap := range []struct {
		name   string
		target *os.File
	}{
		{name: "helper", target: helper},
		{name: "helper directory", target: helperDirectory},
	} {
		present, inspectErr := invokeDarwinACLHelper(ctx, helperPath, bootstrap.target)
		if inspectErr != nil {
			return fmt.Errorf("refusing identity %s %s: descriptor ACL inspection bootstrap failed for %s: %w", kind, path, bootstrap.name, inspectErr)
		}
		if present {
			return fmt.Errorf("refusing identity %s %s: descriptor ACL inspection bootstrap %s has an extended ACL", kind, path, bootstrap.name)
		}
		if err := assertDarwinACLHelperUnchanged(helperPath, helperInfo, helperDirectoryPath, helperDirectoryInfo); err != nil {
			return fmt.Errorf("refusing identity %s %s: descriptor ACL inspection bootstrap changed: %w", kind, path, err)
		}
	}

	present, inspectErr := invokeDarwinACLHelper(ctx, helperPath, target)
	if err := assertDarwinACLHelperUnchanged(helperPath, helperInfo, helperDirectoryPath, helperDirectoryInfo); err != nil {
		return fmt.Errorf("refusing identity %s %s: descriptor ACL inspection failed: %w", kind, path, err)
	}
	if inspectErr != nil {
		return fmt.Errorf("refusing identity %s %s: descriptor ACL inspection failed: %w", kind, path, inspectErr)
	}
	if present {
		return fmt.Errorf("refusing identity %s %s: extended ACLs are not allowed", kind, path)
	}
	return nil
}

func invokeDarwinACLHelper(ctx context.Context, helperPath string, target *os.File) (bool, error) {
	command := exec.CommandContext(ctx, helperPath)
	command.WaitDelay = darwinACLWaitDelay
	command.Stdin = target
	command.Env = []string{}
	var stdout, stderr boundedProtocolCapture
	command.Stdout = &stdout
	command.Stderr = &stderr
	runErr := command.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return false, fmt.Errorf("inspection timed out")
	}

	exitCode := 0
	if runErr != nil {
		var exitError *exec.ExitError
		if !errors.As(runErr, &exitError) {
			return false, runErr
		}
		exitCode = exitError.ExitCode()
	}
	if !stdout.exceeded && !stderr.exceeded && exitCode == 0 && stdout.String() == darwinACLClearResult && stderr.String() == "" {
		return false, nil
	}
	if !stdout.exceeded && !stderr.exceeded && exitCode == darwinACLPresentExit && stdout.String() == darwinACLPresentResult && stderr.String() == "" {
		return true, nil
	}
	return false, fmt.Errorf("inspection returned an invalid result (exit %d)", exitCode)
}

func resolveDarwinACLHelperPath() (string, error) {
	path := darwinACLHelperPathOverride
	if path == "" {
		executable, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("resolve zeroned executable: %w", err)
		}
		executable, err = filepath.EvalSymlinks(executable)
		if err != nil {
			return "", fmt.Errorf("resolve zeroned executable symlinks: %w", err)
		}
		path = filepath.Join(filepath.Dir(executable), darwinACLHelperName)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve ACL helper path: %w", err)
	}
	realDirectory, err := filepath.EvalSymlinks(filepath.Dir(absPath))
	if err != nil {
		return "", fmt.Errorf("resolve ACL helper directory: %w", err)
	}
	return filepath.Join(realDirectory, filepath.Base(absPath)), nil
}

func openValidatedDarwinACLHelperDirectory(path string) (*os.File, os.FileInfo, error) {
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("secure no-follow helper directory open failed: %w", err)
	}
	directory := os.NewFile(uintptr(descriptor), path)
	if directory == nil {
		_ = unix.Close(descriptor)
		return nil, nil, fmt.Errorf("bind ACL helper directory descriptor")
	}
	info, err := directory.Stat()
	if err != nil {
		_ = directory.Close()
		return nil, nil, fmt.Errorf("inspect ACL helper directory: %w", err)
	}
	if err := validateDarwinACLHelperDirectoryStat(directory, info); err != nil {
		_ = directory.Close()
		return nil, nil, err
	}
	return directory, info, nil
}

func validateDarwinACLHelperDirectoryStat(directory *os.File, info os.FileInfo) error {
	if !info.IsDir() {
		return fmt.Errorf("ACL helper directory is not a directory")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("ACL helper directory must not be group/world writable")
	}
	var status unix.Stat_t
	if err := unix.Fstat(int(directory.Fd()), &status); err != nil {
		return fmt.Errorf("inspect ACL helper directory owner: %w", err)
	}
	if status.Uid != 0 && status.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("ACL helper directory is not owned by root or the current user")
	}
	return nil
}

func openValidatedDarwinACLHelper(path string) (*os.File, os.FileInfo, error) {
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("secure no-follow helper open failed: %w; run make darwin-acl-helper", err)
	}
	helper := os.NewFile(uintptr(descriptor), path)
	if helper == nil {
		_ = unix.Close(descriptor)
		return nil, nil, fmt.Errorf("bind ACL helper descriptor")
	}
	info, err := helper.Stat()
	if err != nil {
		_ = helper.Close()
		return nil, nil, fmt.Errorf("inspect ACL helper: %w", err)
	}
	if err := validateDarwinACLHelperStat(helper, info); err != nil {
		_ = helper.Close()
		return nil, nil, err
	}
	var magic [4]byte
	if _, err := helper.ReadAt(magic[:], 0); err != nil {
		_ = helper.Close()
		return nil, nil, fmt.Errorf("read ACL helper Mach-O magic: %w", err)
	}
	if _, ok := fatMachOMagics[magic]; !ok {
		_ = helper.Close()
		return nil, nil, fmt.Errorf("ACL helper is not a universal Mach-O executable")
	}
	return helper, info, nil
}

func validateDarwinACLHelperStat(helper *os.File, info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("ACL helper is not a regular file")
	}
	if info.Mode().Perm() != darwinACLHelperMode {
		return fmt.Errorf("ACL helper must have exact mode 0555")
	}
	if info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return fmt.Errorf("ACL helper must not have setuid, setgid, or sticky bits")
	}
	if info.Size() <= 0 || info.Size() > maxDarwinACLHelperBytes {
		return fmt.Errorf("ACL helper size is outside the trusted bound")
	}
	var status unix.Stat_t
	if err := unix.Fstat(int(helper.Fd()), &status); err != nil {
		return fmt.Errorf("inspect ACL helper owner and links: %w", err)
	}
	if status.Uid != 0 && status.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("ACL helper is not owned by root or the current user")
	}
	if status.Nlink != 1 {
		return fmt.Errorf("ACL helper must have exactly one hard link")
	}
	return nil
}

func assertDarwinACLHelperUnchanged(path string, before os.FileInfo, directoryPath string, directoryBefore os.FileInfo) error {
	if err := assertDarwinACLHelperDirectoryUnchanged(directoryPath, directoryBefore); err != nil {
		return err
	}
	after, afterInfo, err := openValidatedDarwinACLHelper(path)
	if err != nil {
		return err
	}
	defer after.Close()
	if !os.SameFile(before, afterInfo) || before.Size() != afterInfo.Size() || before.Mode() != afterInfo.Mode() || !before.ModTime().Equal(afterInfo.ModTime()) {
		return fmt.Errorf("ACL helper changed while descriptor inspection ran")
	}
	return assertDarwinACLHelperDirectoryUnchanged(directoryPath, directoryBefore)
}

func assertDarwinACLHelperDirectoryUnchanged(path string, before os.FileInfo) error {
	after, afterInfo, err := openValidatedDarwinACLHelperDirectory(path)
	if err != nil {
		return err
	}
	defer after.Close()
	if !os.SameFile(before, afterInfo) || before.Mode() != afterInfo.Mode() || !before.ModTime().Equal(afterInfo.ModTime()) {
		return fmt.Errorf("ACL helper directory changed while descriptor inspection ran")
	}
	return nil
}

type boundedProtocolCapture struct {
	buffer   bytes.Buffer
	exceeded bool
}

func (capture *boundedProtocolCapture) Write(value []byte) (int, error) {
	remaining := maxDarwinACLProtocolBytes - capture.buffer.Len()
	if remaining > 0 {
		toWrite := len(value)
		if toWrite > remaining {
			toWrite = remaining
		}
		_, _ = capture.buffer.Write(value[:toWrite])
	}
	if len(value) > remaining {
		capture.exceeded = true
	}
	return len(value), nil
}

func (capture *boundedProtocolCapture) String() string {
	return capture.buffer.String()
}
