package internal_seat

import "context"

type InternalSeater interface {
	GetSeats(ctx context.Context, venueId int) ([]Seat, error)
}

type Seat struct {
	SeatID      int    `db:"id"`
	SectionCode string `db:"section_code"`
	Row         string `db:"row"`
	SeatNumber  string `db:"seat_number"`
}

func (s *Seat) GetSection() string {
	return s.SectionCode
}

func (s *Seat) GetRow() string {
	return s.Row
}

func (s *Seat) GetSeatNumber() string {
	return s.SeatNumber
}
