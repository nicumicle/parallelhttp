FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o app cmd/service/main.go

# ============================
# Prepare final image
# ============================
FROM gcr.io/distroless/static

COPY --from=builder /app/app .
COPY --from=builder /app/static ./static

# Expose API port
EXPOSE 8080

CMD ["./app"]
