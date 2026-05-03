FROM golang:1.23-alpine AS build
ENV GOTOOLCHAIN=auto

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/payment-service ./cmd/payment-service

FROM alpine:3.20

WORKDIR /app
COPY --from=build /bin/payment-service /usr/local/bin/payment-service
COPY migrations ./migrations
EXPOSE 50051
CMD ["payment-service"]
