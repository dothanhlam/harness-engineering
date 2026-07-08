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

// AuditGeneratedCode scans the directory for security risks and forbidden patterns based on provided rules.
// Goroutine-safe: stateless, read-only filesystem operations.
func AuditGeneratedCode(directory string, ignoreList []string, rules map[string]string) error {
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
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		code := string(content)
		lowerCode := strings.ToLower(code)

		// Check generic rules map
		for pattern, errMsg := range rules {
			if strings.Contains(lowerCode, strings.ToLower(pattern)) || strings.Contains(code, pattern) {
				auditErr = fmt.Errorf("file %s %s", path, errMsg)
				return fmt.Errorf("audit failure")
			}
		}

		return nil
	})
	return auditErr
}

// RunTests executes tests dynamically based on language detection. If forceRegression is true, it iterates over all subdirectories (Go specific).
// Goroutine-safe: spawns an independent child process.
func RunTests(targetDir string, forceRegression bool, ignoreList []string) *TestResult {
	// Language detection
	isNode := false
	if _, err := os.Stat(filepath.Join(targetDir, "package.json")); err == nil {
		isNode = true
	}
	isPython := false
	if _, err := os.Stat(filepath.Join(targetDir, "pytest.ini")); err == nil {
		isPython = true
	} else if _, err := os.Stat(filepath.Join(targetDir, "requirements.txt")); err == nil {
		isPython = true
	}

	var cmd *exec.Cmd
	if isNode {
		cmd = exec.Command("npm", "run", "test")
		cmd.Dir = targetDir
	} else if isPython {
		cmd = exec.Command("pytest")
		cmd.Dir = targetDir
	} else {
		// Fallback to Go
		args := []string{"test", "-v"}

		if forceRegression {
			entries, err := os.ReadDir(targetDir)
			if err != nil {
				return &TestResult{Output: []byte(fmt.Sprintf("failed to read dir %s: %v", targetDir, err)), Err: err}
			}

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
					args = append(args, fmt.Sprintf("./%s/%s/...", filepath.Clean(targetDir), entry.Name()))
					hasTargets = true
				}
			}

			if !hasTargets {
				return &TestResult{Output: []byte("No test targets"), Err: nil}
			}
		} else {
			args = append(args, fmt.Sprintf("./%s/...", filepath.Clean(targetDir)))
		}
		cmd = exec.Command("go", args...)
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
