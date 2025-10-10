FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o wishlist-service .

FROM gcr.io/distroless/static-debian11
WORKDIR /app
COPY --from=builder /app/wishlist-service .
EXPOSE 3002
CMD ["./wishlist-service"]
