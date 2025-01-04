package musicbrainz

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

const ( 
	UserAgent  = "Music Meta/1.0 (client@test.com)" 
 	BaseURL = "https://musicbrainz.org/ws/2"
)

func FetchMusicBrainz(apiEndpoint, mbID string) (*map[string]interface{}, error) {
	client := resty.New()
	client.SetHeader("User-Agent", UserAgent)
	client.SetHeader("Accept", "application/json")

	fmt.Printf("Fetching MusicBrainz Artist Data for %s\n", mbID)
	fmt.Printf("URL: %s/%s/%s\n", BaseURL, apiEndpoint, mbID)

	// Define a generic map to capture the full response
	var fullResponse map[string]interface{}
	resp, err := client.R().
		SetQueryParam("fmt", "json").
		SetResult(&fullResponse).
		Get(fmt.Sprintf("%s/%s/%s", BaseURL, apiEndpoint, mbID))

	fmt.Println("Response Body:", resp.String())

	if err != nil {
		return nil, fmt.Errorf("error making request to MusicBrainz: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("error response from MusicBrainz: %s", resp.Status())
	}

	return &fullResponse, nil
}
