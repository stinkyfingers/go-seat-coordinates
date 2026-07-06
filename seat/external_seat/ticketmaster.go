package external_seat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const MAPS_API_URL = "https://mapsapi.tmol.io/maps/geometry/3/event/%s/placeDetailNoKeys?useHostGrids=true&app=PRD2663_EDP_NA&sectionLevel=true&systemId=HOST"

type TicketmasterSeatResponse struct {
	Pages []Page `json:"pages"`
}
type Page struct {
	Segments []Segment `json:"segments"`
}
type Segment struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	SegmentCategory string    `json:"segmentCategory"`
	Segments        []Segment `json:"segments,omitempty"`
	PlacesNoKeys    []Place   `json:"placesNoKeys,omitempty"`
}

type Place []interface{}

type TicketMaster struct {
}

func NewTicketMaster() *TicketMaster {
	return &TicketMaster{}
}

func (t *TicketMaster) GetSeatCoordinates(ctx context.Context, eventId string) ([]Seat, error) {
	data, err := fetchEventJson(ctx, eventId)
	if err != nil {
		return nil, err
	}
	var response TicketmasterSeatResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	var seats []Seat
	for _, page := range response.Pages {
		for _, segment := range page.Segments {
			err := segment.dig("", "", &seats)
			if err != nil {
				return nil, err
			}
		}
	}
	return seats, nil
}

func fetchEventJson(ctx context.Context, eventId string) ([]byte, error) {
	url := fmt.Sprintf(MAPS_API_URL, eventId)
	cli := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "seat-mapping-3/1.0")
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch event JSON: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (s *Segment) dig(sectionName, rowName string, seats *[]Seat) error {
	switch s.SegmentCategory {
	case "SECTION":
		for _, segment := range s.Segments {
			segment.dig(s.Name, "", seats)
		}
	case "ROW":
		for _, place := range s.PlacesNoKeys {
			seatNumber, x, y, err := place.getSeatCoordinates()
			if err != nil {
				return fmt.Errorf("error getting seat coordinates for place %v: %v", place, err)
			}
			*seats = append(*seats, Seat{
				SeatNumber:  seatNumber,
				SeatRow:     s.Name,
				SeatSection: sectionName,
				X:           x,
				Y:           y,
			})
		}

	case "COMPOSITE":
		fallthrough
	default:
		for _, segment := range s.Segments {
			segment.dig("", "", seats)
		}
	}
	return nil
}

func (p *Place) getSeatCoordinates() (string, float64, float64, error) {
	if len(*p) < 4 {
		return "", 0, 0, fmt.Errorf("invalid place data")
	}
	seatNumber, ok := (*p)[1].(string)
	if !ok {
		return "", 0, 0, fmt.Errorf("invalid seat number: %v", (*p)[0])
	}
	if seatNumber == "" {
		return "", 0, 0, fmt.Errorf("seat number is empty")
	}

	x, ok := (*p)[2].(float64)
	if !ok {
		return "", 0, 0, fmt.Errorf("invalid x coordinate: %v", (*p)[2])
	}

	y, ok := (*p)[3].(float64)
	if !ok {
		return "", 0, 0, fmt.Errorf("invalid y coordinate: %v", (*p)[3])
	}

	return seatNumber, x, y, nil
}

func (s *Seat) GetSection() string {
	return s.SeatSection
}

func (s *Seat) GetRow() string {
	return s.SeatRow
}

func (s *Seat) GetSeatNumber() string {
	return s.SeatNumber
}
