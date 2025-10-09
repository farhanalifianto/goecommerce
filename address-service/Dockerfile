FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o address-service .

FROM gcr.io/distroless/static-debian11
WORKDIR /app
COPY --from=builder /app/address-service .
EXPOSE 3002
CMD ["./address-service"]
