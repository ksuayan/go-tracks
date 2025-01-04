package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"math/rand"
	"os"
	"syscall"
	"time"
)

// getFileCreationDate retrieves the creation date (birth time) of a file using syscall.Stat_t.
func GetFileCreationDate(path string) (time.Time, error) {
	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err != nil {
		return time.Time{}, err
	}
	// Use Ctimespec as a reliable fallback for file creation time
	return time.Unix(stat.Ctimespec.Sec, stat.Ctimespec.Nsec), nil
}

func GetUniqueID() string {
  rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%d_%d", time.Now().Unix(), rand.Intn(1000)) // Unix timestamp + random number
}

// Generate SHA-256 Hash of a File
func GetFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func ParseNumWorkers(input string) int {
	var numWorkers int
	_, err := fmt.Sscanf(input, "%d", &numWorkers)
	if err != nil {
		fmt.Printf("Invalid number of workers: %s. Defaulting to 1.\n", input)
		return 1
	}

	// Ensure numWorkers is within a reasonable range
	if numWorkers < 1 {
		fmt.Printf("Number of workers cannot be less than 1. Defaulting to 1.\n")
		return 1
	} else if numWorkers > 64 {
		fmt.Printf("Number of workers is too high (%d). Limiting to 64.\n", numWorkers)
		return 64
	}
	return numWorkers
}

// SafeGet is a helper function that safely accesses nested fields in a map.
func SafeGet(myMap map[string]interface{}, keys ...string) (interface{}, bool) {
	current := myMap
	for _, key := range keys {
		if current == nil {
			return nil, false
		}
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			return nil, false
		}
	}
	return current, true
}