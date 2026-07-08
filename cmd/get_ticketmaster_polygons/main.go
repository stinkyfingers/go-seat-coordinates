package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/stinkyfingers/go-seat-coordinates/polygon"
)

var (
	venue = flag.String("v", "", "Venue code, matching a filename in ./svgs (e.g. atb for svgs/atb.svg)")
)

func main() {
	flag.Parse()
	if *venue == "" {
		log.Fatal("Venue code is required")
	}

	svgPath := fmt.Sprintf("svgs/%s.svg", *venue)
	sections, err := polygon.ParseFile(svgPath)
	if err != nil {
		log.Fatalf("parsing %s: %v", svgPath, err)
	}

	filename := filepath.Join("results", fmt.Sprintf("polygons_%s.json", *venue))
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(sections)
	if err != nil {
		log.Fatal(err)
	}
}
