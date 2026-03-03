FROM golang:1.26-alpine AS builder

WORKDIR /app 

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o backend ./cmd/api

FROM alpine

COPY --from=builder /app/backend /backend

EXPOSE 8080

CMD ["/backend"]