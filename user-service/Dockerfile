FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o user-service .

FROM gcr.io/distroless/static-debian11
WORKDIR /app
COPY --from=builder /app/user-service .
EXPOSE 3001
CMD ["./user-service"]
