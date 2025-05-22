# syntax=docker/dockerfile:1

# Étape 1 : Build binaire Go statique
FROM golang:1.24 AS builder

WORKDIR /app

# Copier les fichiers Go du projet
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compiler en statique, version optimisée pour Docker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rss-feed-scrapper

# Étape 2 : Image finale minimaliste
FROM gcr.io/distroless/static

WORKDIR /

COPY --from=builder /app/rss-feed-scrapper /rss-feed-scrapper

EXPOSE 8080

ENTRYPOINT ["/rss-feed-scrapper"]
