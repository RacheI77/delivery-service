# Delivery Service API

A RESTful delivery order management API built with Go. It allows clients to place orders (with distance calculated via Google Maps API), take orders, and list orders with pagination.

## Tech Stack

- **Language:** Go 1.24
- **Database:** MySQL 8.0
- **Deployment:** Docker & Docker Compose

## Quick Start

### Prerequisites

- Docker & Docker Compose installed
- A Google Maps API key (with Distance Matrix API enabled)

> **Windows users:** `start.sh` is a bash script. On Windows, run it from **Git Bash** (or WSL) — it is not directly executable from cmd/PowerShell. Docker itself is cross-platform, so the rest of the stack works the same on Windows.

### 1. Run the start script

```bash
./start.sh
```

A single command brings up the whole environment. `start.sh` automates everything:

1. If `.env` does not exist, it is **auto-created** from `.env.example`, so no manual setup is required
2. If the `GOOGLE_MAPS_API_KEY` is still the placeholder, the script **interactively prompts** you to enter a real key (or press Enter to skip)
3. Tear down any existing containers/volumes
4. Build and start the MySQL database and the API service
5. Wait until the API is ready on port 8080

The API will be available at `http://localhost:8080`.

> **Note:** You can also set your actual Google Maps API key at any time by editing `.env`:

```bash
GOOGLE_MAPS_API_KEY="YOUR_ACTUAL_API_KEY"
```

Then re-run `./start.sh` (which rebuilds the containers) or `docker-compose up -d` (which keeps your data) to apply the change.

> **Note:** The application also reads environment variables. If an environment variable is set, it takes precedence over the value in the `.env` file. This is useful for CI/CD or when you don't want to commit secrets.

> **Security:** The `.env` file is **not** committed to version control (see `.gitignore`) and is **excluded** from the Docker build context (see `.dockerignore`). Secrets are injected into the container at runtime via environment variables defined in `docker-compose.yml`, so they are never baked into the image.

### 2. Stop the service

```bash
docker-compose down
```

## Configuration

Configuration can be provided either through a `.env` config file in the project root, or through environment variables (which take precedence). The `.env` file uses `KEY=VALUE` lines; lines starting with `#` are comments.

### How the `.env` file is used in Docker deployment

The `.env` file is the **single source of truth** for configuration, with two distinct roles during `docker-compose up`:

1. **Variable substitution** – Docker Compose automatically reads the `.env` file to fill in the `${VAR}` placeholders in `docker-compose.yml` (e.g. `${PORT}`, `${DB_PASSWORD}` for the `db` service and port mappings).
2. **Container injection** – The `app` service references the same file via `env_file: .env`, so every key-value pair becomes an environment variable inside the application container. The only override is `DB_HOST=db`, because inside the container the MySQL server is reached via the Compose service name `db`, not `localhost`.

So you only ever edit **one** file (`.env`) — whether you run the app natively with `go run ./cmd/server` (which reads `.env` directly) or inside Docker.

| Config Key / Env Variable | Default | Description |
|---------------------------|---------|-------------|
| `PORT` | `8080` | HTTP port the API listens on |
| `DB_HOST` | `localhost` | MySQL host |
| `DB_PORT` | `3306` | MySQL port |
| `DB_USER` | `root` | MySQL user |
| `DB_PASSWORD` | `root` | MySQL password |
| `DB_NAME` | `delivery_db` | MySQL database name |
| `GOOGLE_MAPS_API_KEY` | *(required)* | Google Maps API key with Distance Matrix API enabled |
| `GOOGLE_MAPS_API_URL` | `https://maps.googleapis.com/maps/api/distancematrix/json` | Google Maps Distance Matrix API endpoint |

## API Endpoints

### 1. Place an Order

Creates a new order and calculates the distance between origin and destination using the Google Maps Distance Matrix API.

- **Method:** `POST`
- **URL:** `/orders`
- **Request Body:**

```json
{
    "origin": ["22.3193", "114.1694"],
    "destination": ["22.3964", "114.1095"]
}
```

- **Success Response:** `HTTP 200`

```json
{
    "id": 1,
    "distance": 12345,
    "status": "UNASSIGNED"
}
```

- **Error Response:** `HTTP 400` (invalid coordinates)

```json
{
    "error": "INVALID_COORDINATES"
}
```

**Validation rules:**
- `origin` and `destination` must be arrays of exactly **two strings**
- Latitude must be between -90 and 90
- Longitude must be between -180 and 180

### 2. Take an Order

Marks an order as `TAKEN`. An order can only be taken once — concurrent requests to take the same order will result in only one success.

- **Method:** `PATCH`
- **URL:** `/orders/:id`
- **Request Body:**

```json
{
    "status": "TAKEN"
}
```

- **Success Response:** `HTTP 200`

```json
{
    "status": "SUCCESS"
}
```

- **Error Responses:**

| HTTP Code | Body | Description |
|-----------|------|-------------|
| `400` | `{"error": "INVALID_ORDER_ID"}` | Order ID is not a valid positive integer |
| `400` | `{"error": "INVALID_STATUS"}` | Request body status is not `"TAKEN"` |
| `404` | `{"error": "ORDER_NOT_FOUND"}` | Order does not exist |
| `409` | `{"error": "ORDER_ALREADY_TAKEN"}` | Order has already been taken |

### 3. List Orders

Returns a paginated list of orders.

- **Method:** `GET`
- **URL:** `/orders?page=:page&limit=:limit`
- **Example:** `/orders?page=1&limit=10`

- **Success Response:** `HTTP 200`

```json
[
    {
        "id": 1,
        "distance": 12345,
        "status": "UNASSIGNED"
    },
    {
        "id": 2,
        "distance": 6789,
        "status": "TAKEN"
    }
]
```

- **Empty Result:** `HTTP 200` with empty array

```json
[]
```

- **Error Response:** `HTTP 400` (invalid pagination parameters)

```json
{
    "error": "INVALID_PAGINATION_PARAMETERS"
}
```

**Validation rules:**
- `page` must be a valid integer starting from 1
- `limit` must be a valid positive integer
- `limit` is capped at a maximum of `100` to protect the database from excessively large queries

**Error Responses:**

| HTTP Code | Body | Description |
|-----------|------|-------------|
| `400` | `{"error": "INVALID_PAGINATION_PARAMETERS"}` | `page` or `limit` is not a valid positive integer |
| `404` | `{"error": "NOT_FOUND"}` | The requested path does not exist |
| `405` | `{"error": "METHOD_NOT_ALLOWED"}` | The HTTP method is not supported for the path |

## Project Structure

```
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── db/
│   └── init.sql                 # Database schema initialization
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration loading (.env + env vars)
│   ├── handler/
│   │   ├── order_handler.go     # HTTP handlers
│   │   └── order_handler_test.go
│   ├── model/
│   │   └── order.go             # Data models
│   ├── repository/
│   │   └── order_repository.go  # Database access layer
│   └── service/
│       └── distance_service.go  # Google Maps API integration
├── .env.example                  # Example configuration file
├── docker-compose.yml
├── Dockerfile
├── start.sh
└── README.md
```

## Testing

### Unit tests

Run all unit tests (no external dependencies required):

```bash
go test ./...
```

### Integration tests

Integration tests exercise the repository layer against a real MySQL instance, including a concurrency test that verifies the "take order" race-condition requirement (exactly one concurrent request succeeds).

They are automatically **skipped** if the database is not reachable, so they are safe to run in any environment. To run them against a live database (e.g. the one started by `docker-compose`), set the following environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `TEST_DB_HOST` | `localhost` | MySQL host |
| `TEST_DB_PORT` | `3306` | MySQL port |
| `TEST_DB_USER` | `root` | MySQL user |
| `TEST_DB_PASSWORD` | `root` | MySQL password |
| `TEST_DB_NAME` | `delivery_db` | MySQL database name |

## Race Condition Handling

The "take order" endpoint uses an atomic SQL update to prevent race conditions:

```sql
UPDATE orders SET status = 'TAKEN' WHERE id = ? AND status = 'UNASSIGNED'
```

When multiple concurrent requests try to take the same order, MySQL's row-level locking ensures only one `UPDATE` affects a row. The others see `RowsAffected == 0` and receive a `409 CONFLICT` response.