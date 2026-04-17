# URL Shortener

A high-performance URL shortening service built with a microservices architecture using Go, gRPC, Kafka, Redis, and PostgreSQL.

## Architecture

Soon...
## Services

### API Gateway
- REST HTTP server — single entry point for all client requests
- Routes requests to internal gRPC services
- JWT authentication middleware
- Rate limiting middleware

### Auth Service
- User registration and login
- JWT token generation and validation
- Redis for token storage and blacklisting

### URL Service
- URL shortening with slug generation
- Redirect handling
- Redis caching for fast lookups
- Publishes redirect events to Kafka

### Stats Service (legacy)
- Reads redirect events from Kafka
- Stores raw redirect data in PostgreSQL

### Stats Service New
- Reads redirect events from Kafka with **batch processing**
- In-memory aggregation before database write
- Bot detection via User-Agent parsing
- Device detection (mobile/tablet/desktop)
- Aggregated statistics stored in PostgreSQL
- **110x throughput improvement** over legacy service (150 → 16,400 events/sec)

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
│   ├── stats-service/      # Legacy stats (raw redirects)
│   └── stats-service-new/  # Optimized stats (aggregated)
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
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/sign-up` | Register new user |
| POST | `/auth/sign-in` | Login, returns JWT tokens |
| POST | `/auth/refresh` | Refresh access token |

### URLs
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/urls` | Create short URL |
| GET | `/:slug` | Redirect to original URL |
| GET | `/urls` | Get all user URLs |
| DELETE | `/urls/:id` | Delete URL |

### Stats
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/stats/:url_id` | Get click statistics for URL |

## Performance

Stats service was optimized through several iterations:

| Optimization | Throughput |
|-------------|------------|
| Baseline (single INSERT per event) | 20 events/sec |
| Kafka partitions 1 → 8 | 160 events/sec |
| Batch processing (500 events/batch) | 4,000 events/sec |
| Aggregated schema (UPSERT) | 16,400 events/sec |

Key optimizations:
- **Kafka partitioning** — 8 partitions with consumer pool
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
  port: :50053

stats-service-new:
  port: :50054
```

`.env`:
```
DB_PASSWORD=your_password
JWT_SECRET=your_secret
```
