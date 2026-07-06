package internal_seat

import (
	"context"
	"database/sql"
)

type Hexagon struct {
	DB *sql.DB
}

func NewHexagon(db *sql.DB) *Hexagon {
	return &Hexagon{
		DB: db,
	}
}

func (h *Hexagon) GetSeats(ctx context.Context, venueId int) ([]Seat, error) {
	qs := `select s.id, s.section_code, s.row, s.seat_number from seats s 
join sections ss on ss.id = s.section_id 
where ss.venue_id = $1`
	rows, err := h.DB.QueryContext(ctx, qs, venueId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seats []Seat
	for rows.Next() {
		var seat Seat
		if err := rows.Scan(&seat.SeatID, &seat.SectionCode, &seat.Row, &seat.SeatNumber); err != nil {
			return nil, err
		}
		seats = append(seats, seat)
	}
	return seats, nil
}
