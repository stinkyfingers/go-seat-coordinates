FROM golang:latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .
RUN mkdir -p /app/results
ENTRYPOINT ["sh", "-c", "cd /app && /app/main -e \"$EVENT_ID\" -v \"$VENUE_ID\" --db \"$DB_CONN_STRING\" --input-source \"$INPUT_SOURCE\" --skip-llm \"$SKIP_LLM\""]