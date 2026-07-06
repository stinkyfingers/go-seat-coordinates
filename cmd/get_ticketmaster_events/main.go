package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	attractionsFile = "ticketmaster_data/attractions.json"
	tmEventsURL     = "https://app.ticketmaster.com/discovery/v2/events.json"
	pageSize        = 200
)

var projectId = flag.Int("p", 0, "Project ID")

func main() {
	flag.Parse()
	if *projectId == 0 {
		log.Fatal("project_id is required")
	}

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment")
	}

	apiKey := os.Getenv("TM_API_KEY")
	if apiKey == "" {
		log.Fatal("TM_API_KEY not found in environment or .env")
	}

	attractionId, err := loadAttractionID(attractionsFile, *projectId)
	if err != nil {
		log.Fatal(err)
	}

	eventIds, err := fetchAllEventIDs(apiKey, attractionId)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(eventIds)
}

func loadAttractionID(path string, projectId int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("attractions.json not found: %s: %w", path, err)
	}
	var attractions map[string]string
	if err := json.Unmarshal(data, &attractions); err != nil {
		return "", fmt.Errorf("parsing attractions.json: %w", err)
	}
	attractionId, ok := attractions[strconv.Itoa(projectId)]
	if !ok || attractionId == "" {
		return "", fmt.Errorf("no attraction ID found for project %d in %s", projectId, path)
	}
	return attractionId, nil
}

type eventsPage struct {
	Embedded struct {
		Events []tmEvent `json:"events"`
	} `json:"_embedded"`
	Page struct {
		TotalPages    int `json:"totalPages"`
		TotalElements int `json:"totalElements"`
	} `json:"page"`
}

type tmEvent struct {
	URL      string `json:"url"`
	Embedded struct {
		Attractions []struct {
			ID string `json:"id"`
		} `json:"attractions"`
	} `json:"_embedded"`
}

func fetchPage(apiKey, attractionId string, page int) (*eventsPage, error) {
	params := url.Values{}
	params.Set("apikey", apiKey)
	params.Set("attractionId", attractionId)
	params.Set("size", strconv.Itoa(pageSize))
	params.Set("page", strconv.Itoa(page))

	reqURL := tmEventsURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for page %d: %w", page, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TM API request failed for page %d: %w", page, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response for page %d: %w", page, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TM API returned HTTP %d for page %d", resp.StatusCode, page)
	}

	var payload eventsPage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("TM API returned invalid JSON for page %d: %w", page, err)
	}
	return &payload, nil
}

func eventIDFromURL(eventURL string) string {
	trimmed := strings.TrimRight(eventURL, "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1]
}

func isHomeEvent(event tmEvent, attractionId string) bool {
	attractions := event.Embedded.Attractions
	return len(attractions) > 0 && attractions[0].ID == attractionId
}

func extractHomeEventIDs(payload *eventsPage, attractionId string) (home, away []string) {
	for _, event := range payload.Embedded.Events {
		eventId := eventIDFromURL(event.URL)
		if eventId == "" {
			continue
		}
		if isHomeEvent(event, attractionId) {
			home = append(home, eventId)
		} else {
			away = append(away, eventId)
		}
	}
	return home, away
}

func fetchAllEventIDs(apiKey, attractionId string) ([]string, error) {
	var eventIds []string
	page := 0

	for {
		fmt.Printf("  Fetching page %d... ", page)
		payload, err := fetchPage(apiKey, attractionId, page)
		if err != nil {
			return nil, err
		}

		totalPages := payload.Page.TotalPages
		if totalPages == 0 {
			totalPages = 1
		}

		if page == 0 {
			fmt.Printf("(%d total events, %d pages)\n", payload.Page.TotalElements, totalPages)
		}

		home, away := extractHomeEventIDs(payload, attractionId)
		eventIds = append(eventIds, home...)
		fmt.Printf("  %d home, %d away skipped (running total: %d)\n", len(home), len(away), len(eventIds))

		page++
		if page >= totalPages {
			break
		}
	}

	return eventIds, nil
}
