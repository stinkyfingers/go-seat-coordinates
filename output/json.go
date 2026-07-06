package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/stinkyfingers/go-seat-coordinates/seat"
)

func WriteJSONToFile(seats []seat.Seat, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	b, err := json.MarshalIndent(seats, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	_, err = f.Write(b)
	if err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}
	return nil
}
