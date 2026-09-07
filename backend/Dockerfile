# Multi-stage build для backend-сервиса QADAM.

FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /qadam-server ./cmd/server

FROM alpine:3.20
RUN adduser -D -g '' qadam
WORKDIR /app
COPY --from=build /qadam-server /usr/local/bin/qadam-server
COPY --from=build /app/migrations ./migrations
RUN chown -R qadam:qadam /app
USER qadam
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/qadam-server"]
