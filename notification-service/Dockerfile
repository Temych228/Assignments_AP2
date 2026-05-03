FROM golang:1.23-alpine AS build
ENV GOTOOLCHAIN=auto

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download || true
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /bin/notification-service ./cmd/notification-service

FROM alpine:3.20

WORKDIR /app
COPY --from=build /bin/notification-service /usr/local/bin/notification-service
CMD ["notification-service"]
