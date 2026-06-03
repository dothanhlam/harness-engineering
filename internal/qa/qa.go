package qa

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TestResult holds the output and error from a test run.
type TestResult struct {
	Output []byte
	Err    error
}

// AuditGeneratedCode scans the directory for security risks and forbidden patterns in .go files.
// Goroutine-safe: stateless, read-only filesystem operations.
func AuditGeneratedCode(directory string) error {
	var auditErr error
	_ = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		code := string(content)
		lowerCode := strings.ToLower(code)

		if strings.Contains(code, "\"os/exec\"") || strings.Contains(code, "exec.Command") {
			auditErr = fmt.Errorf("file %s invokes forbidden package os/exec", path)
			return fmt.Errorf("audit failure")
		}
		if strings.Contains(code, "rm -rf") {
			auditErr = fmt.Errorf("file %s contains destructive terminal command 'rm -rf'", path)
			return fmt.Errorf("audit failure")
		}
		if strings.Contains(code, "os.Remove(") || strings.Contains(code, "os.RemoveAll(") || strings.Contains(code, "os.Rename(") {
			auditErr = fmt.Errorf("file %s contains unauthorized filesystem manipulation", path)
			return fmt.Errorf("audit failure")
		}
		if strings.Contains(lowerCode, "password =") || strings.Contains(lowerCode, "secret =") || strings.Contains(lowerCode, "aws_access_key") {
			auditErr = fmt.Errorf("file %s contains potential hardcoded credentials", path)
			return fmt.Errorf("audit failure")
		}

		return nil
	})
	return auditErr
}

// RunTests executes `go test -v` on the given pattern and returns the combined result.
// Goroutine-safe: spawns an independent child process.
func RunTests(pattern string) *TestResult {
	var cmd *exec.Cmd
	if strings.HasPrefix(pattern, "./workspace/") {
		subPattern := "./" + strings.TrimPrefix(pattern, "./workspace/")
		cmd = exec.Command("go", "test", "-v", subPattern)
		cmd.Dir = "workspace"
	} else if pattern == "./workspace/..." {
		cmd = exec.Command("go", "test", "-v", "./...")
		cmd.Dir = "workspace"
	} else {
		cmd = exec.Command("go", "test", "-v", pattern)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return &TestResult{Output: out.Bytes(), Err: err}
}

// CountGeneratedLines counts total lines across all .go files in the given directory tree.
// Goroutine-safe: read-only filesystem walk.
func CountGeneratedLines(dir string) int {
	var count int
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			content, err := os.ReadFile(path)
			if err == nil {
				count += len(strings.Split(string(content), "\n"))
			}
		}
		return nil
	})
	return count
}
