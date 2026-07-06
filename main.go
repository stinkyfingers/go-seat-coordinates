package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // To register the driver.
	"github.com/stinkyfingers/go-seat-coordinates/mapping"
	"github.com/stinkyfingers/go-seat-coordinates/output"
	"github.com/stinkyfingers/go-seat-coordinates/seat"
	"github.com/stinkyfingers/go-seat-coordinates/seat/external_seat"
	"github.com/stinkyfingers/go-seat-coordinates/seat/internal_seat"
)

var (
	venueId     = flag.Int("v", 0, "Venue ID")
	eventId     = flag.String("e", "", "Event ID")
	databaseURL = flag.String("db", "", "Database connection string (optional, can also be set via DB_CONN_STRING environment variable)")
	skipLLM     = flag.Bool("skip-llm", false, "Skip LLM section mapping (optional, default false)")
)

func main() {
	ctx := context.Background()

	// flags
	flag.Parse()
	if *venueId == 0 {
		log.Fatal("Venue ID is required")
	}
	if *eventId == "" {
		log.Fatal("Event ID is required")
	}

	// .env
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on system environment")
	}

	// db
	dbString := os.Getenv("DB_CONN_STRING")
	if *databaseURL != "" {
		dbString = *databaseURL
	}
	if dbString == "" {
		log.Fatal("Database connection string is required (set via -db flag or DB_CONN_STRING environment variable)")
	}
	db, err := sql.Open("postgres", dbString)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// run process
	err = process(ctx, db, *venueId, *eventId, *skipLLM)
	if err != nil {
		log.Fatalf("Error processing venue: %v", err)
	}

}

func process(ctx context.Context, db *sql.DB, venueId int, eventId string, skipLLM bool) error {
	// get seats from internal database - hexagon
	hex := internal_seat.NewHexagon(db)
	internalSeats, err := hex.GetSeats(ctx, venueId)
	if err != nil {
		log.Fatalf("Error getting seats: %v", err)
	}
	fmt.Println("# internal seats", len(internalSeats))

	// get seats (w/ coordinates) from external source - ticketmaster
	tm := external_seat.NewTicketMaster()
	externalSeats, err := tm.GetSeatCoordinates(ctx, eventId)
	if err != nil {
		return err
	}
	fmt.Println("# external seats", len(externalSeats))

	// convert to Seaters
	var internal []seat.Seater
	var external []seat.Seater
	for _, s := range internalSeats {
		internal = append(internal, &s)
	}
	for _, s := range externalSeats {
		external = append(external, &s)
	}

	// map internal to external
	internalToExternalMapping, missingSections, err := mapping.MapInternalToExternal(external, internal, skipLLM)
	if err != nil {
		return fmt.Errorf("mapping internal to external seats: %w", err)
	}
	fmt.Println("Unmapped sections:", missingSections)
	fmt.Println("Mapped seats:", len(internalToExternalMapping))

	// write seat coordinates to JSON file
	filename := fmt.Sprintf(filepath.Join("results", "output_%d_%s.json"), venueId, eventId)
	err = output.WriteJSONToFile(internalToExternalMapping, filename)
	if err != nil {
		return fmt.Errorf("writing JSON to file: %w", err)
	}

	return nil
}
