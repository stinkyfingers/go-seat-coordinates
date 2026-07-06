package internal_seat

import (
	"context"
	"database/sql"
)

type DemandAnalytics struct {
	DB *sql.DB
}

func NewDemandAnalytics(db *sql.DB) *DemandAnalytics {
	return &DemandAnalytics{
		DB: db,
	}
}

func (h *DemandAnalytics) GetSeats(ctx context.Context, venueId int) ([]Seat, error) {
	qs := `select vcs.seat_id as id, vcs.seat_section as section_code, vcs.seat_row as row, vcs.seat_number from venue.venue_configuration_seat vcs 
join venue.venue_configuration vc on vc.venue_configuration_id = vcs.venue_configuration_id 
where vc.is_current = B'1'
and vc.venue_id = $1`
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
