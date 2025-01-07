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
	RootDir    			 string    `bson:"rootDir"`	
	SubDir				   string    `bson:"subDir"`
	FileName         string    `bson:"fileName"`
	FileExtension    string    `bson:"fileExtension"`
	CreationDate     time.Time `bson:"creationDate"`
	ModificationDate time.Time `bson:"modificationDate"`
	Title            string    `bson:"title"`
	Artist           string    `bson:"artist"`
	Album            string    `bson:"album"`
	AlbumArtist			 string    `bson:"albumArtist"`
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

	totalFiles := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return nil // Continue processing other files
		}

		if !info.IsDir() {

			fileExt := strings.ToLower(filepath.Ext(info.Name()))

			if IsAudioFile(fileExt) {
				dirPath := filepath.Dir(path)
				subDir := utils.GetSubDir(path, root, info.Name())
				fileName := info.Name()
				modDate := info.ModTime()

				log.Printf("dir: %s, audio file: %s", subDir, fileName)

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

				ffprobeData, err := ffprobe.GetFFProbe(fullpath)
				if err != nil {
					log.Printf("Error getting ffprobe for %s: %v", info.Name(), err)
				} 

				// Send FileInfo to the channel
				fileChan <- FileInfo{
					RootDir:				 root,
					SubDir:        	 subDir,
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
					FFProbe: 			   *ffprobeData,
					AlbumArtist: 		 utils.SafeGetTagValue(ffprobeData.Format.Tags,	"album_artist"),
				}

				totalFiles++

			} else {
				log.Printf("Skipping non-audio file: %s", path)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Error walking the path: %v", err)
	}

	log.Printf("Total audio files scanned: %d", totalFiles)

	doneChan <- err
}

func UpdateDatabase(db *mongo.Database, fileChan <-chan FileInfo, doneChan <-chan error) error {
	collection := db.Collection("tracks")
	insertedCount := 0

	for {
		select {
		case file, ok := <-fileChan:
			if !ok {
				// fileChan closed; wait for doneChan
				err := <-doneChan
				log.Printf("Total files inserted/updated: %d\n", insertedCount)
				return err
			}

			// Update the database
			filter := bson.M{"rootDir": file.RootDir, "subDir": file.SubDir, "fileName": file.FileName}
			update := bson.M{"$set": file}
			_, err := collection.UpdateOne(context.Background(), 
				filter, 
				update, 
				options.Update().SetUpsert(true))

			if err != nil {
				log.Printf("Error updating database for %s: %v", file.FileName, err)
			} else {
				insertedCount++
			}

		case err := <-doneChan:
			return err
		}
	}
}

func ScanDirectoryAndUpdateDB(root string, db *mongo.Database) error {
	fileChan := make(chan FileInfo, 1000) // Buffered channel for FileInfo
	doneChan := make(chan error, 1)     // Channel for signaling completion

	// Start scanning in a separate goroutine
	go ScanDirectoryAsync(root, fileChan, doneChan)

	// Update the database while scanning
	return UpdateDatabase(db, fileChan, doneChan)
}
