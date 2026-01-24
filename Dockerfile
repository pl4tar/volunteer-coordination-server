# ---- build stage ----
FROM golang:1.25.3-alpine AS builder

WORKDIR /src

# git может понадобиться для go modules
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /src/cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/app .

# ---- runtime stage ----
FROM alpine:3.21

WORKDIR /app

# сертификаты часто нужны для https-запросов наружу
RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/app /app/app
COPY --from=builder /src/config /app/config

EXPOSE 4949
CMD ["/app/app"]
