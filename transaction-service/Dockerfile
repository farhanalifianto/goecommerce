FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o transaction-service .

FROM gcr.io/distroless/static-debian11
WORKDIR /app
COPY --from=builder /app/transaction-service .
EXPOSE 3003
CMD ["./transaction-service"]
