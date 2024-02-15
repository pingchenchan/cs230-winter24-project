# CS230 Go Monitor

This project is a simple monitoring application built with Go, which interacts with Docker and Redis to manage container lifecycles and provide real-time status updates.

## Features

- Start a new `nginx` Docker container every 10 seconds.
- Stop the oldest running Docker container every 20 seconds.
- Store and retrieve container IDs from Redis.
- Provide a RESTful endpoint to fetch the list of running containers.

## Prerequisites

Before you begin, ensure you have met the following requirements:

- Docker is installed and running.
- Redis server is accessible.
- You have a basic understanding of Docker and containerization.

## Installing CS230 Go Monitor

To install CS230 Go Monitor, follow these steps:

Linux and macOS:

```bash
git clone https://github.com/yourusername/cs230-go-monitor.git
cd cs230-go-monitor
```

## Using CS230 Go Monitor
To use CS230 Go Monitor, start the services using docker-compose:

```bash
docker-compose up --build
```

This command will start the Go application along with Redis and any other defined services in the `docker-compose.yml` file.

## API Reference
- `GET /containers`: Returns a JSON list of all running container IDs.