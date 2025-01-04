package coverart

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ksuayan/go-tracks/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// convertToJPEG converts an image file to JPEG format
func ConvertToJPEG(inputFile, outputFile string) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("error decoding image: %w", err)
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	options := &jpeg.Options{Quality: 90}
	if err := jpeg.Encode(outFile, img, options); err != nil {
		return fmt.Errorf("error encoding image to JPEG: %w", err)
	}

	return nil
}

func ExtractCoverArt(db *mongo.Database, filePath string, outputDir string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
  uniqueID := utils.GetUniqueID()
	// Generate a unique filename by appending timestamp and random number
	tempFile := filepath.Join(outputDir, "temp", fmt.Sprintf("cover_%s.jpg", uniqueID))

	// Run the appropriate command to extract cover art
	var cmd *exec.Cmd
	switch ext {
	case ".flac":
		cmd = exec.Command("metaflac", fmt.Sprintf("--export-picture-to=%s", tempFile), filePath)
	case ".m4a", ".mp4", ".alac", ".mp3":
		fmt.Printf(">>> ffmpeg (.m4a): Extracting cover art from %s\n", filePath)
		cmd = exec.Command("ffmpeg", "-loglevel", "quiet", "-i", filePath, "-an", "-frames:v", "1", "-update", "1", tempFile)
	default:
    fmt.Printf(">>> ffmpeg (default): Extracting cover art from %s\n", filePath)
		cmd = exec.Command("ffmpeg", "-loglevel", "quiet", "-i", filePath, "-an", "-frames:v", "1", "-update", "1", tempFile)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("error extracting cover art: %w", err)
	}

  
	// Generate a hash for the cover art file
	hash, err := utils.GetFileHash(tempFile)
	if err != nil {
		return "", "", err
	}

	// Use the hash to create a two-level directory structure
	level1 := hash[:2]   // First two characters
	level2 := hash[2:4]  // Next two characters
	targetDir := filepath.Join(outputDir, level1, level2)

	// Create the directories if they don't exist
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("error creating directories: %w", err)
	}

	// Move the file to the final directory
	hashedFilePath := filepath.Join(targetDir, fmt.Sprintf("%s.jpg", hash))
	if err := os.Rename(tempFile, hashedFilePath); err != nil {
		return "", "", fmt.Errorf("error renaming file: %w", err)
	}

	coverArtCollection := db.Collection("coverart")
	// Save cover art metadata in the `coverart` collection
	_, err = coverArtCollection.UpdateOne(
		context.TODO(),
		bson.M{"hash": hash},
		bson.M{
			"$set": bson.M{
				"hash":    hash,
				"filePath": hashedFilePath,
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return "", "", fmt.Errorf("error updating coverart collection: %w", err)
	}

	return hash, hashedFilePath, nil
}

// GetFilePath generates the file path for a given hash value.
// It uses a two-level directory structure based on the first 4 characters of the hash.
func GetCoverArtPathFromHash(outputDir, hash string) (string, error) {
	if len(hash) < 4 {
		return "", fmt.Errorf("hash must be at least 4 characters long")
	}

	// Extract the first two levels of the directory structure
	level1 := hash[:2]   // First two characters
	level2 := hash[2:4]  // Next two characters

	// Construct the full file path
	filePath := filepath.Join(outputDir, level1, level2, fmt.Sprintf("%s.jpg", hash))

	return filePath, nil
}