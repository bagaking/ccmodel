package execcmd

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	sessionPrefix   = "ccmodel-"
	sessionHashSize = 6
)

var nonNameChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func deriveSessionName(dir string) (string, error) {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = wd
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	base := filepath.Base(abs)
	if base == "." || base == string(os.PathSeparator) || base == "" {
		base = "root"
	}
	base = sanitizeSegment(base)
	h := sha1.Sum([]byte(abs))
	segment := hex.EncodeToString(h[:])
	if len(segment) > sessionHashSize {
		segment = segment[:sessionHashSize]
	}
	return fmt.Sprintf("%s%s-%s", sessionPrefix, base, segment), nil
}

func sanitizeSegment(name string) string {
	sanitized := nonNameChars.ReplaceAllString(name, "-")
	if sanitized == "" {
		sanitized = "session"
	}
	return sanitized
}

func parseTimeSafe(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts
	}
	return time.Time{}
}
