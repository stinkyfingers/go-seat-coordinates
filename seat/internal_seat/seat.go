package internal_seat

import "github.com/stinkyfingers/go-seat-coordinates/seat"

type InternalSeater interface {
	seat.Seater
	GetSeats(venueId int) ([]Seat, error)
}

type Seat struct {
	SeatID      int    `db:"id"`
	SectionCode string `db:"section_code"`
	Row         string `db:"row"`
	SeatNumber  string `db:"seat_number"`
}
