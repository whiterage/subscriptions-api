FROM golang:1.25-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/subscriptions ./cmd/subscriptions

FROM alpine:3.22

WORKDIR /app
RUN addgroup -S app && adduser -S app -G app
COPY --from=build /app/subscriptions /app/subscriptions
COPY docs /app/docs

EXPOSE 8080
USER app
CMD ["/app/subscriptions"]
