package external_seat

import "github.com/stinkyfingers/go-seat-coordinates/seat"

type ExternalSeater interface {
	seat.Seater
	GetSeatCoordinates() ([]Seat, error)
}


type Seat struct {
	SeatNumber  string  `json:"seatNumber"`
	SeatRow     string  `json:"seatRow"`
	SeatSection string  `json:"seatSection"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
}
