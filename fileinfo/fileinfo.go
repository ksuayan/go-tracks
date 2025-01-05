package fileinfo

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ksuayan/go-tracks/ffprobe"
	"github.com/ksuayan/go-tracks/utils"

	"github.com/wtolson/go-taglib"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FileInfo struct {
	FilePath				 string    `bson:"filePath"`
	DirectoryPath    string    `bson:"directoryPath"`	
	FileName         string    `bson:"fileName"`
	FileExtension    string    `bson:"fileExtension"`
	CreationDate     time.Time `bson:"creationDate"`
	ModificationDate time.Time `bson:"modificationDate"`
	Title            string    `bson:"title"`
	Artist           string    `bson:"artist"`
	Album            string    `bson:"album"`
	AlbumArtist			 string    `bson:"album_artist"`
	Year             int       `bson:"year"`
	Genre            string    `bson:"genre"`
	Bitrate          int       `bson:"bitrate"`
	Samplerate       int       `bson:"samplerate"`
	Channels         int       `bson:"channels"`
	Length           time.Duration       `bson:"length"`
	Track            int       `bson:"track"`
	Status           string    `bson:"status"`
	CoverArt         string    `bson:"coverArt"`
	CoverArtHash     string    `bson:"coverArtHash"`
	FileHash         string    `bson:"fileHash"`
	FFProbe					 ffprobe.FFProbe   `bson:"ffprobe"`
}

// List of known audio file extensions
var audioExtensions = []string{
	".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma", ".m4a", ".aiff", ".alac", ".opus",
}

// Checks if the file has a known audio extension
func IsAudioFile(extension string) bool {
	extension = strings.ToLower(extension)
	for _, ext := range audioExtensions {
		if ext == extension {
			return true
		}
	}
	return false
}
func ScanDirectoryAsync(root string, fileChan chan<- FileInfo, doneChan chan<- error) {
	defer close(fileChan) // Close the channel when done

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			fileExt := strings.ToLower(filepath.Ext(info.Name()))
			if IsAudioFile(fileExt) {
				dirPath := filepath.Dir(path)
				fileName := info.Name()
				modDate := info.ModTime()

				// Get creation date
				creationDate, err := utils.GetFileCreationDate(path)
				if err != nil {
					creationDate = time.Time{}
				}

				// Open the audio file and read metadata
				fullpath := filepath.Join(dirPath, fileName)
				audioMetadata, err := taglib.Read(fullpath)
				if err != nil {
					log.Printf("Error reading metadata for %s: %v", fullpath, err)
					return nil
				}
				defer audioMetadata.Close()

				// Generate file hash
				fileHash, err := utils.GetFileHash(fullpath)
				if err != nil {
					log.Printf("Error generating file hash for %s: %v", fullpath, err)
					return nil
				}

				// Send FileInfo to the channel
				fileChan <- FileInfo{
					FilePath:        fullpath,
					DirectoryPath:   dirPath,
					FileName:        fileName,
					FileExtension:   fileExt,
					CreationDate:    creationDate,
					ModificationDate: modDate,
					Title:           audioMetadata.Title(),
					Artist:          audioMetadata.Artist(),
					Album:           audioMetadata.Album(),
					Year:            audioMetadata.Year(),
					Genre:           audioMetadata.Genre(),
					Bitrate:         audioMetadata.Bitrate(),
					Samplerate:      audioMetadata.Samplerate(),
					Channels:        audioMetadata.Channels(),
					Length:          audioMetadata.Length(),
					Track:           audioMetadata.Track(),
					Status:          "new",
					CoverArt:				 "",
					CoverArtHash:    "",
					FileHash:        fileHash,
				}
			}
		}
		return nil
	})

	doneChan <- err // Send error (or nil) when done
}

func UpdateDatabase(db *mongo.Database, fileChan <-chan FileInfo, doneChan <-chan error) error {
	collection := db.Collection("tracks")
	for {
		select {
		case file, ok := <-fileChan:
			if !ok {
				// fileChan closed; wait for doneChan
				err := <-doneChan
				return err
			}

			// Enrich with FFProbe data
			ffprobeData, err := ffprobe.GetFFProbe(file.FilePath)
			if err != nil {
				log.Printf("Error getting ffprobe output for %s: %v", file.FileName, err)
			} else {
				file.FFProbe = *ffprobeData
				tags := ffprobeData.Format.Tags
				file.AlbumArtist = utils.SafeGetTagValue(tags,	"album_artist")
			}

			// Update the database
			filter := bson.M{"filePath": file.FilePath}
			update := bson.M{"$set": file}
			_, err = collection.UpdateOne(context.Background(), filter, update, options.Update().SetUpsert(true))
			if err != nil {
				log.Printf("Error updating database for %s: %v", file.FileName, err)
			}
		case err := <-doneChan:
			return err
		}
	}
}

func ScanDirectoryAndUpdateDB(root string, db *mongo.Database) error {
	fileChan := make(chan FileInfo, 100) // Buffered channel for FileInfo
	doneChan := make(chan error, 1)     // Channel for signaling completion

	// Start scanning in a separate goroutine
	go ScanDirectoryAsync(root, fileChan, doneChan)

	// Update the database while scanning
	return UpdateDatabase(db, fileChan, doneChan)
}
