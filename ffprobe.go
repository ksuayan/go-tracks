package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// FFProbe represents the structure of ffprobe JSON output
type FFProbe struct {
	Streams []Stream `bson:"streams"`
	Format  Format   `bson:"format"`
}

// Stream represents an individual stream in ffprobe output
type Stream struct {
	Index        int    `bson:"index"`
	CodecName    string `bson:"codec_name"`
	CodecType    string `bson:"codec_type"`
	BitRate      string `bson:"bit_rate,omitempty"`
	SampleRate   string `bson:"sample_rate,omitempty"`
	Channels     int    `bson:"channels,omitempty"`
	ChannelLayout string `bson:"channel_layout,omitempty"`
	Width        int    `bson:"width,omitempty"`
	Height       int    `bson:"height,omitempty"`
	Duration     string `bson:"duration,omitempty"`
}

// Format represents the format section in ffprobe output
type Format struct {
	Filename string            `bson:"filename"`
	Duration string            `bson:"duration"`
	BitRate  string            `bson:"bit_rate"`
	Size     string            `bson:"size"`
	Tags     map[string]string `bson:"tags"`
}

// getFFProbe runs ffprobe on the input file and parses the JSON output
func getFFProbe(inputFile string) (*FFProbe, error) {
	fmt.Printf(">>> ffprobe: Running ffprobe on %s\n", inputFile)
	cmd := exec.Command("ffprobe", "-i", inputFile, "-show_format", "-show_streams", "-print_format", "json", "-v", "quiet")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running ffprobe: %w", err)
	}

	var ffprobe FFProbe
	if err := json.Unmarshal(output, &ffprobe); err != nil {
		return nil, fmt.Errorf("error parsing ffprobe JSON: %w", err)
	}

	return &ffprobe, nil
}
