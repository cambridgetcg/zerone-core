//go:build tok_learning && unix

package cross_stack_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// A path may become a FIFO after Lstat. Open without waiting for a writer, then
// let the caller check the descriptor before reading. Root retains confinement.
func tokLearningOpenReadFile(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}

func TestToKLearningReceiptsFIFOReplacement(t *testing.T) {
	if os.Getenv("TOK_LEARNING_RECEIPTS_FIFO_CHILD") != "1" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, executable, "-test.run=^TestToKLearningReceiptsFIFOReplacement$", "-test.v")
		command.Env = append(os.Environ(), "TOK_LEARNING_RECEIPTS_FIFO_CHILD=1")
		output, err := command.CombinedOutput()
		if ctx.Err() != nil {
			t.Fatalf("receipt open blocked after regular-to-FIFO replacement: %v: %s", ctx.Err(), output)
		}
		if err != nil || !strings.Contains(string(output), "FIFO replacement rejected before read") {
			t.Fatalf("FIFO replacement subprocess: %v: %s", err, output)
		}
		return
	}

	directory := t.TempDir()
	name := "head.json"
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	replaced := false
	data, err := tokLearningReadFileWithOpenHook(root, name, 1024, func() {
		// The reader has already checked the regular inode. Keep that inode
		// alive and replace its name with a FIFO having no reader or writer.
		if err := root.Rename(name, name+".old"); err != nil {
			t.Fatal(err)
		}
		if err := unix.Mkfifo(path, 0600); err != nil {
			t.Fatal(err)
		}
		replaced = true
	})
	if !replaced || err == nil || err.Error() != "receipt file changed during open" || data != nil {
		t.Fatalf("expected opened-descriptor rejection after FIFO replacement, got %q, %v", data, err)
	}
	t.Log("FIFO replacement rejected before read")
}

func TestToKLearningReceiptsOpenedDescriptorChecks(t *testing.T) {
	for _, scenario := range []string{"regular", "replaced_inode", "grown_inode", "escaping_symlink"} {
		t.Run(scenario, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "head.json")
			if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			data, err := tokLearningReadFileWithOpenHook(root, "head.json", 2, func() {
				switch scenario {
				case "replaced_inode", "escaping_symlink":
					if err := root.Rename("head.json", "old.json"); err != nil {
						t.Fatal(err)
					}
					if scenario == "escaping_symlink" {
						outside := filepath.Join(t.TempDir(), "outside.json")
						if err := os.WriteFile(outside, []byte("{}"), 0600); err != nil {
							t.Fatal(err)
						}
						if err := os.Symlink(outside, path); err != nil {
							t.Fatal(err)
						}
					} else if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
						t.Fatal(err)
					}
				case "grown_inode":
					if err := os.WriteFile(path, []byte("{} "), 0600); err != nil {
						t.Fatal(err)
					}
				}
			})
			if scenario == "regular" {
				if err != nil || string(data) != "{}" {
					t.Fatalf("bounded regular file: %q, %v", data, err)
				}
			} else if err == nil || data != nil {
				t.Fatalf("accepted changed file: %q, %v", data, err)
			} else if scenario != "escaping_symlink" && err.Error() != "receipt file changed during open" {
				t.Fatalf("expected rejection before read: %v", err)
			}
		})
	}
}
