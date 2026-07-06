package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/stinkyfingers/go-seat-coordinates/seat/external_seat"
)

/*
Fetch seats from Ticketmaster for a given event ID and print the number of seats found.
*/

var (
	eventId = flag.String("e", "", "Event ID")
)

func main() {
	flag.Parse()
	ctx := context.Background()
	ticketmaster := external_seat.NewTicketMaster()
	seats, err := ticketmaster.GetSeatCoordinates(ctx, *eventId)
	if err != nil {
		panic(err)
	}
	fmt.Println("S", len(seats))
}
