FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/backend/services/api-gateway

COPY ../../../backend/shared/ ./shared/

RUN go test ./... -v

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/bin/api-gateway \
    ./cmd/rest


FROM alpine:latest AS runner

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/bin/api-gateway .

COPY --from=builder /app/backend/.env ./
COPY --from=builder /app/backend/configs/ ./configs/

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8090

CMD ["./api-gateway"]