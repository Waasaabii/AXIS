package axis

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func mustJSONClone[T any](input T) T {
	raw, _ := json.Marshal(input)
	var out T
	_ = json.Unmarshal(raw, &out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

func findBinaryInPath(binary string) string {
	path, err := exec.LookPath(binary)
	if err != nil {
		return ""
	}
	return path
}

func compilePattern(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	return regexp.Compile(pattern)
}

func createNodeID(source, canonical string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:%s", source, canonical)))
	return hex.EncodeToString(sum[:])[:12]
}

func pathOrResolved(configPath, binary string) string {
	if strings.Contains(binary, "/") || strings.HasPrefix(binary, ".") {
		return filepath.Clean(filepath.Join(filepath.Dir(configPath), binary))
	}
	return findBinaryInPath(binary)
}
