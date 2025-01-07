package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
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
		log.Printf("Invalid number of workers: %s. Defaulting to 1.\n", input)
		return 1
	}

	// Ensure numWorkers is within a reasonable range
	if numWorkers < 1 {
		log.Printf("Number of workers cannot be less than 1. Defaulting to 1.\n")
		return 1
	} else if numWorkers > 64 {
		log.Printf("Number of workers is too high (%d). Limiting to 64.\n", numWorkers)
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

// SafeGetTagValue is a helper function that safely retrieves a tag value from a map.
func SafeGetTagValue( tags map[string] string, tagName string) (string){
	tagValue, ok := tags[tagName]
	if !ok {
		log.Printf("Error: %s not found or not a string", tagName)
	}
	return tagValue
}

func GetSubDir(filePath, rootDir, fileName string) string {
	// Ensure the filePath starts with rootDir
	if !strings.HasPrefix(filePath, rootDir) {
		return "" // Return empty if filePath doesn't start with rootDir
	}

	// Remove rootDir from filePath
	relativePath := strings.TrimPrefix(filePath, rootDir)

	// Ensure relativePath starts with a path separator
	if len(relativePath) > 0 && relativePath[0] == '/' {
		relativePath = relativePath[1:]
	}

	// Remove the fileName from relativePath
	if strings.HasSuffix(relativePath, fileName) {
		relativePath = strings.TrimSuffix(relativePath, fileName)
	}

	// Remove any trailing slash from the subdirectory
	return strings.TrimSuffix(relativePath, "/")
}

func SecondsToDuration(seconds float64) (int, int, int, float64) {
	// Convert the total seconds into whole hours, minutes, and seconds
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60

	// Extract the fractional part of the seconds
	fractionalSeconds := math.Mod(seconds, 1)

	return hours, minutes, secs, fractionalSeconds
}