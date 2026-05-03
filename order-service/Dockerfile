FROM golang:1.23-alpine AS build
ENV GOTOOLCHAIN=auto

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/order-service ./cmd/order-service

FROM alpine:3.20

WORKDIR /app
COPY --from=build /bin/order-service /usr/local/bin/order-service
COPY migrations ./migrations
EXPOSE 8080 9090
CMD ["order-service"]
