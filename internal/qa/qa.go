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
func AuditGeneratedCode(directory string, ignoreList []string) error {
	var auditErr error
	_ = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			for _, ign := range ignoreList {
				if info.Name() == ign {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
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

// RunTests executes `go test -v` on the subdirectories of the given base directory, skipping ignored ones.
// Goroutine-safe: spawns an independent child process.
func RunTests(baseDir string, ignoreList []string) *TestResult {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return &TestResult{Output: []byte(fmt.Sprintf("failed to read dir %s: %v", baseDir, err)), Err: err}
	}

	args := []string{"test", "-v"}
	hasTargets := false

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ignored := false
		for _, ign := range ignoreList {
			if entry.Name() == ign {
				ignored = true
				break
			}
		}
		if !ignored {
			args = append(args, fmt.Sprintf("./%s/%s/...", filepath.Clean(baseDir), entry.Name()))
			hasTargets = true
		}
	}

	if !hasTargets {
		return &TestResult{Output: []byte("No test targets"), Err: nil}
	}

	cmd := exec.Command("go", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
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
