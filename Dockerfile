FROM golang:1.26-alpine AS base
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod ./

FROM base AS deps
COPY . .
RUN go mod download

FROM deps AS api
RUN go build -o /out/api ./cmd/api
ENTRYPOINT ["/out/api"]

FROM deps AS bot
RUN go build -o /out/bot ./cmd/bot
ENTRYPOINT ["/out/bot"]

FROM deps AS worker
RUN go build -o /out/worker ./cmd/worker
ENTRYPOINT ["/out/worker"]
