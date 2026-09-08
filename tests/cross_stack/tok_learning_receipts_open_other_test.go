//go:build tok_learning && !unix

package cross_stack_test

import (
	"fmt"
	"os"
	"runtime"
)

// Keep the opt-in pilot present on unsupported platforms, but fail explicitly
// instead of falling back to a blocking open or reporting a no-tests pass.
func tokLearningOpenReadFile(root *os.Root, name string) (*os.File, error) {
	return nil, fmt.Errorf("ToK learning receipt nonblocking open unsupported on %s", runtime.GOOS)
}
