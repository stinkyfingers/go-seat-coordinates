package seat

type Seater interface {
	GetSection() string
	GetRow() string
	GetSeatNumber() string
}

type Seaters []Seater

type Seat struct {
	InternalID  int     `json:"internalId,omitempty"`
	SeatNumber  string  `json:"seatNumber"`
	SeatRow     string  `json:"seatRow"`
	SeatSection string  `json:"seatSection"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
}
