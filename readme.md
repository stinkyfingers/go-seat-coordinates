Basic Process:

- Get internal seats w/ sections - either from DA or Hex databases (by flag, below)
- Get TM data from many events and aggregate unique seat coordinates
    - ad hoc cmd to get TM event ids
- Map internal -> external using Bedrock
- Merge internal seats with external coordinates
- Write


RUN: 

`go run main.go -v <venue_id> -e <event_id>` 
- optionally pass `--db <DB_CONN STRING>`
- optionally pass `--input-source <'da' or 'hex'>` - assure you have either a DA or Hex DB correct in the --db flag or DB_CONN_STRING .env value.
- venue is the Hexagon venue ID. 
- event is a TicketMaster Event ID for the team/venue.


DOCKER:

Build:

```
docker build -t go-seat-coordinates .
```

Run:

```
docker run \
    -e EVENT_ID=0F00635BEE7872E5 \
    -e VENUE_ID=7 \
    -e DB_CONN_STRING='<DB_CONN_STRING>' \
    -e INPUT_SOURCE=<'hex' or 'da' | default hex> \
    -v ~/.aws:/root/.aws:ro \
    -v "$(pwd)/results:/app/results" \
    go-seat-coordinates
```


OUTPUT:

Output results are stored in ./results and named `output_<venue_id>_<event_id>.json`


TicketMaster Data

./ticketmaster_data stores data that we've collected from TicketMaster APIs
- tm_events and attractions.json use DA projectIDs to denote team - needs update
