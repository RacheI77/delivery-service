package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type DistanceCalculator interface {
	GetDistance(origin, destination []string) (int, error)
}

type GoogleDistanceService struct {
	APIKey  string
	BaseURL string
}

type distanceMatrixResponse struct {
	Rows []struct {
		Elements []struct {
			Distance struct {
				Value int `json:"value"`
			} `json:"distance"`
			Status string `json:"status"`
		} `json:"elements"`
	} `json:"rows"`
	Status string `json:"status"`
}

func (g *GoogleDistanceService) GetDistance(origin, destination []string) (int, error) {
	apiKey := g.APIKey
	if apiKey == "" {
		return 0, fmt.Errorf("GOOGLE_MAPS_API_KEY is missing")
	}

	origStr := fmt.Sprintf("%s,%s", origin[0], origin[1])
	destStr := fmt.Sprintf("%s,%s", destination[0], destination[1])

	endpoint := fmt.Sprintf(
		"%s?origins=%s&destinations=%s&key=%s",
		g.BaseURL, url.QueryEscape(origStr), url.QueryEscape(destStr), apiKey,
	)

	resp, err := http.Get(endpoint)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Google Maps API returned status code %d", resp.StatusCode)
	}

	var data distanceMatrixResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	if data.Status != "OK" || len(data.Rows) == 0 || len(data.Rows[0].Elements) == 0 || data.Rows[0].Elements[0].Status != "OK" {
		return 0, fmt.Errorf("failed to fetch valid distance from Google API")
	}

	return data.Rows[0].Elements[0].Distance.Value, nil
}
