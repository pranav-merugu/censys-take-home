# Censys Take Home Assignment

A distributed key-value store built with Go, featuring a gRPC backend service and a REST API gateway.

## Architecture

```
┌─────────────────────────────────────────────┐
│         Client (HTTP/REST)                   │
└──────────────────┬──────────────────────────┘
                   │
                   │ HTTP REST
                   ▼
         ┌──────────────────────┐
         │  API Service         │
         │  Port: 8080          │
         │  Framework: Gin      │
         └──────────┬───────────┘
                    │
                    │ gRPC
                    ▼
         ┌──────────────────────┐
         │  KV Service          │
         │  Port: 50051         │
         │  Storage: In-Memory  │
         │  Concurrency: RWMutex│
         └──────────────────────┘
```

## How to Run the Project

### Option 1: Run Locally with Go

Run the KV Store server:

```bash
cd kv-service
go run .
```

Run the REST API server:

```bash
cd api-service
go run .
```

### Option 2: Docker

In the project root folder, run:

```bash
docker compose build
docker compose up
```

The REST APIs will be available at `localhost:8080`.

### Available Endpoints

- `GET /kv/:key` - Retrieve a value by key
- `POST /kv` - Store a key-value pair (Request body: `{"key": "...", "value": "..."}`)
- `DELETE /kv/:key` - Delete a key-value pair

## Testing Instructions

Run all tests (unit tests and integration tests):

```bash
go test ./...
```

Run unit tests for individual services:

```bash
# KV Store server
cd kv-service
go test -v

# REST API server
cd api-service
go test -v
```

Run integration tests:

```bash
cd tests
go test -v
```

## Assumptions Made During Development

- **Data persistence is not required** - The key-value store uses an in-memory map, so data is lost when the service restarts
- **No authentication required** - The API endpoints are publicly accessible without any authentication or authorization mechanisms

## Future Improvements

- **Add data persistence** - Implement disk-based storage or integrate with a database to persist data across restarts
- **Add authentication and authorization** - Secure the API endpoints with API keys or OAuth to control access

## Implementation Details

For the REST API server, I used the Gin framework. The server creates 3 endpoints that communicate with the storage server using gRPC.

For the KV Store server, the key-value store logic is handled with a simple map structure. The server is a gRPC server and accepts messages specified in the `proto/kvstore.proto` file.

To handle concurrency, the KV Store server uses a read-write mutex, allowing for multiple simultaneous reads or a single write at a time. This avoids race conditions if multiple requests are made simultaneously.
