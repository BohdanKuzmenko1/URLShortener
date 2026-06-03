# URL Shortener

A high-performance URL shortening service built with a microservices architecture using Go, gRPC, Kafka, Redis, and PostgreSQL.

## Architecture
![Architecture.png](backend/docs/Architecture.png)

## Services

### API Gateway
- REST HTTP server — single entry point for all client requests
- Routes requests to internal gRPC services
- JWT authentication middleware

### Auth Service
- User registration and login
- JWT token generation and validation
- Redis for token storage

### URL Service
- URL shortening with slug generation
- Redirect handling
- Redis and LRU caching for fast lookups
- Publishes redirect events to Kafka

### Stats Service
- Reads redirect events from Kafka with **batch processing**
- In-memory aggregation before database write
- Bot detection via User-Agent parsing
- Device detection (mobile/tablet/desktop)
- Aggregated statistics stored in PostgreSQL
- **150x throughput improvement** over legacy service (150 → 23,000 events/sec)

## Tech Stack

| Technology | Usage |
|------------|-------|
| Go | All services |
| gRPC + Protobuf | Inter-service communication |
| REST/HTTP (Gin) | API Gateway |
| PostgreSQL | Persistent storage |
| Redis | Caching, token storage |
| Apache Kafka | Async event streaming |
| Docker / Docker Compose | Containerization |
| golang-migrate | Database migrations |

## Project Structure

```
.
├── proto/                  # Protobuf definitions
├── services/
│   ├── api-gateway/        # REST API gateway
│   ├── auth-service/       # Authentication & authorization
│   ├── url-service/        # URL shortening & redirects
│   ├── stats-service/      # Optimized stats
├── shared/                 # Shared types (JWT claims)
├── deploy/
│   ├── docker-compose.yml
│   └── migrations/         # SQL migrations
└── configs/
    └── config.yml
```

## Getting Started

### Prerequisites

- Docker & Docker Compose
- Go 1.21+

### Run with Docker Compose

```bash
# Clone the repository
git clone https://github.com/BohdanKuzmenko1/URLShortener.git
cd URLShortener

# Copy environment variables
cp .env.example .env

# Start all services
docker compose -f deploy/docker-compose.yml up --build
```

### Run migrations

```bash
migrate -path deploy/migrations -database "postgres://user:password@localhost:5432/urlshortener?sslmode=disable" up
```

## API Endpoints

### Auth
| Method | Endpoint                  | Description               |
|--------|---------------------------|---------------------------|
| POST   | `/api/auth/register`      | Register new user         |
| POST   | `/api/auth/login`         | Login, returns JWT tokens |
| POST   | `/api/auth/refresh-token` | Refresh access token      |
| DELETE | `/api/auth/logout`        | Delete refresh token      |
### URLs
| Method | Endpoint            | Description |
|--------|---------------------|-------------|
| POST | `/api/url/shorten` | Create short URL |
| GET | `/:slug`            | Redirect to original URL |
| GET | `/api/urls`         | Get all user URLs |
### Stats
| Method | Endpoint                   | Description |
|--------|----------------------------|-------------|
| GET | `/api/url-stats/?id=&date` | Get click statistics for URL |

## Performance

Stats service was optimized through several iterations:

| Optimization | Throughput        |
|-------------|-------------------|
| Baseline (single INSERT per event) | ≈20 events/sec     |
| Batch processing (500 events/batch) | ≈4,000 events/sec  |
| Aggregated schema (UPSERT) | ≈23,000 events/sec |

Key optimizations:
- **Batch processing** — collect 500 events then bulk insert
- **In-memory aggregation** — reduce DB writes by grouping `(url_id, date, country, device)`
- **PostgreSQL UPSERT** — single query handles both insert and update
- **Connection pooling** — `SetMaxOpenConns(25)` to prevent DB overload

## Configuration

`configs/config.yml`:

```yaml
postgres:
  host: localhost
  port: 5432
  username: postgres
  dbname: urlshortener
  sslmode: disable

redis:
  address: localhost:6379

api-gateway:
  port: :8080

auth-service:
  port: :50052

url-service:
  port: :50051

stats-service:
  port: :50054
```

`.env`:
```
DB_PASSWORD=your_password
JWT_SECRET=your_secret
```
