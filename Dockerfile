# --- dependency cache ---
FROM golang:1.25-alpine AS base
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download

# --- build all binaries in one layer ---
FROM base AS build
COPY . .
RUN go build -o /out/api ./cmd/api \
 && go build -o /out/bot ./cmd/bot \
 && go build -o /out/worker ./cmd/worker

# --- minimal runtime images (no source, no secrets) ---
FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/api /usr/local/bin/api
ENTRYPOINT ["api"]

FROM alpine:3.21 AS bot
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/bot /usr/local/bin/bot
ENTRYPOINT ["bot"]

FROM alpine:3.21 AS worker
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/worker /usr/local/bin/worker
ENTRYPOINT ["worker"]
