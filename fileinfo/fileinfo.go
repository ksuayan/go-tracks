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


func ScanDirectory(root string) ([]FileInfo, error) {
	var files []FileInfo

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

				// Open the audio file and read the embedded metadata
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

				files = append(files, FileInfo{
					FilePath: 			  fullpath,
					DirectoryPath:    dirPath,
					FileName:         fileName,
					FileExtension:    fileExt,
					CreationDate:     creationDate,
					ModificationDate: modDate,
					Title:            audioMetadata.Title(),
					Artist:           audioMetadata.Artist(),
					Album:            audioMetadata.Album(),
					Year:             audioMetadata.Year(),
					Genre:            audioMetadata.Genre(),
					Bitrate:          audioMetadata.Bitrate(),
					Samplerate:       audioMetadata.Samplerate(),
					Channels:         audioMetadata.Channels(),
					Length:           audioMetadata.Length(),
					Track:            audioMetadata.Track(),
					Status:           "new",
					CoverArtHash: 		"",
					FileHash:         fileHash,
				})
			}
		}
		return nil
	})

	return files, err
}

func ScanDirectoryAndUpdateDB(root string, db *mongo.Database) error {
	files, err := ScanDirectory(root)
	if err != nil {
		return err
	}
	collection := db.Collection("tracks")

	for _, file := range files {

		// Enrich with FFProbe data
		ffprobeData, err := ffprobe.GetFFProbe(file.FilePath)
		if err != nil {
			log.Printf("Error getting ffprobe output for %s: %v", file.FileName, err)
		} else {
			file.FFProbe = *ffprobeData
		}
		// match by file path
		filter := bson.M{"filePath": file.FilePath} // Match by file hash
		update := bson.M{
			"$set": file, // Set all fields
		}
		_, err = collection.UpdateOne(context.Background(), filter, update, options.Update().SetUpsert(true))
		if err != nil {
			log.Printf("Error updating database for %s: %v", file.FileName, err)
			return err
		}
	}
	return nil
}
