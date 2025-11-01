# Censys Take Home Assignment

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

## Usage

### Option 1: Run code

Run the KV Store server:

```bash
cd kv-service
go run .
```

Run the REST API server:

```bash
cd ../api-service
go run .
```

### Option 2: Docker

Note that the Docker container worked on my machine with Go version go1.24.4 darwin/arm64 on an ARM64 architecture Mac.

In the project root folder, run:

```bash
docker compose build
docker compose up
```

## Testing

Once one of the above options is complete, both the server that handles key value storage and the server that handles REST APIs are running. The REST APIs are available at `localhost:8080`

To test that things are working, run the provided test file:

```bash
cd tests
go run .
```

You can also make separate API calls to `localhost:8080`.

The available endpoints are:

- `/kv/:key` GET request
- `/kv` POST request: Request body needs a "key" and "value"
- `/kv/:key` DELETE request

## Implementation

For the REST API server, I used the Gin framework. The server simply creates 3 endpoints. When these endpoints are called, it uses gRPC to communicate with the storage server.

For the KV Store server, the Key-Value store logic is handled with a simple map structure. The server is a gRPC server and accepts messages specified in the `proto/kvstore.proto` file.

To setup the gRPC communication, I followed some guides online as it required many specific commands and files.

To handle concurrency, the KV Store server uses a read write mutex, allowing for multiple simultaneous reads or a single write at a time. This avoids running into issues of race conditions if multiple requests are made simultaneously.

Note that the data in the KV Store server is not persisted.

## Goal

Build a simple decomposed Key-Value store by implementing two services which communicate over gRPC.

The first service should implement a basic JSON Rest API to serve as the primary public interface. This service should then externally communicate with a second service over gRPC, which implement a basic Key-Value store service that can:

1. Store a value at a given key
2. Retrieve the value for a given key
3. Delete a given key

The JSON interface should at a minimum be able to expose and implement these three functions.

You can write this in whichever languages you choose, however Go would be preferred. Ideally, the final result should be built into two separate Docker containers which can be used to run each service independently.
