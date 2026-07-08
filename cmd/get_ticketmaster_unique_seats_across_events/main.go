package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/stinkyfingers/go-seat-coordinates/seat/external_seat"
)

/*
	Gets all seats for all events in <team>_tm_events.json.
	For each event, gets all TM seats. Determines unique events required
	to get all seats.

	Prerequisite: run get_ticketmaster_events/main.go for the team first to generate a file with an array of all of an attraction's eventIds.

	Motivation: I've noticed that, on occassion, an attraction will have different seats for different events. There may be a deeper issue, but this crudely
	determines if a single event will have all of a venue's seats, or if we need to consider multiple events. E.g. ATB offers "Premium Seating",
	seemingly as a separate TM event.
*/

var teamAbbreviation = flag.String("t", "", "Team Abbrevation, e.g. 'atb', 'bao'")

func main() {
	ctx := context.Background()
	flag.Parse()
	if *teamAbbreviation == "" {
		log.Fatal("team abbreviation is required")
	}

	// get tm event ids for attraction
	eventIds, err := getEventIds(*teamAbbreviation)
	if err != nil {
		log.Fatal(err)
	}

	// get each event's seats
	eventSeatsMap := make(map[string][]external_seat.Seat)
	tm := external_seat.NewTicketMaster()
	for _, eventId := range eventIds {
		seats, err := tm.GetSeatCoordinates(ctx, eventId)
		if err != nil {
			log.Fatal(err)
		}
		eventSeatsMap[eventId] = seats
	}
	// log # seats per event
	for eventId, seats := range eventSeatsMap {
		fmt.Println("event ", eventId, " has ", len(seats), " seats.")
	}

	// determine events with unique seats
	filteredEventIdsMap := make(map[string]struct{})
	seatEventMap := make(map[external_seat.Seat]string)
	for eventId, seats := range eventSeatsMap {
		for _, seat := range seats {
			_, ok := seatEventMap[seat]
			if !ok {
				filteredEventIdsMap[eventId] = struct{}{}
				seatEventMap[seat] = eventId
			}
		}
	}

	// compile array of events required to get as many unique seats as possible
	var filteredEventIds []string
	for eventId := range filteredEventIdsMap {
		filteredEventIds = append(filteredEventIds, eventId)
	}
	fmt.Println(filteredEventIds)
}

func getEventIds(teamAbbreviation string) ([]string, error) {
	f, err := os.Open(fmt.Sprintf("%s_tm_events.json", teamAbbreviation))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var eventIds []string
	err = json.NewDecoder(f).Decode(&eventIds)
	if err != nil {
		return nil, err
	}
	return eventIds, nil
}
