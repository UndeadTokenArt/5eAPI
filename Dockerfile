FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/dnd5e .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /out/dnd5e /usr/local/bin/dnd5e
COPY --from=builder /src/static ./static
COPY --from=builder /src/templates ./templates
COPY --from=builder /src/tiles ./tiles

EXPOSE 8080
CMD ["/usr/local/bin/dnd5e"]
