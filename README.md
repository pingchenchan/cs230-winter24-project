# CS230 Monitoring System for Auto-Scaling Applications

This project is a simple monitoring application built with Go, which interacts with Docker and Redis to manage container lifecycles and provide real-time status updates. We also design a simple load balancer and an application to demonstarte the auto-scaling capability of the monitoring system. 

## Features

### Load Balancer
- Load balance requests from users to application servers.
- Check alive servers every 10 seconds.
- Retry disconnected servers every 30 seconds.
- Retry a request 3 times if failed and retry 3 times to pick a different server.
- Report dead servers to the primary monitor every 10 seconds.

### Application


### Monitor
- Start a new `nginx` Docker container every 10 seconds.
- Stop the oldest running Docker container every 20 seconds.
- Store and retrieve container IDs from Redis.
- Provide a RESTful endpoint to fetch the list of running containers.

## System Architecture

TODO

To better understand the context, please refer to [Consistency Model](#consistency-model).

## Prerequisites

Before you begin, ensure you have met the following requirements:

- Docker is installed and running.
- Redis server is accessible.
- You have a basic understanding of Docker and containerization.


## Installation

To install CS230 Go Monitor, follow these steps:

Linux and macOS:

```bash
git clone git@github.com:pingchenchan/cs230-winter24-project.git
cd cs230-winter24-project
```

## Usage
To use CS230 Go Monitor, start the services using docker-compose:

```bash
docker-compose up --build
```

This command will start the Go application along with Redis and any other defined services in the `docker-compose.yml` file.

## Consistency Model

### Context
In this project, the assumption is that we want to scale a service by building a scalable and cost-effective surrounding architecture. The service can be anything once it can be easily replicated. For example, a simple stateless web service like an online calculator to a state-of-the-art scam detector machine learning model.

### Design

TODO

<!-- The first step is to add a load balancer so that we can distribute requests to multiple replicated servers.

The second step is t

To ensure the fault tolerance of the monitoring system, the load balancers and monitors are replicated.


For application itself, it doesn't know anything. The only reponsibility is to serve the user request. For the metric agent running in the background, it also only needs to know the presence of Kakfa. We thus completely isolate the application, making incredibly  -->

### Consistency Model

The reason why we choose to completely divide servers into several zone is to prevent consistency of add servers and delete servers. For example, assume the monitor sends the new servers addresses to all of the load balancers. It will be fine initially. However, if there is a network partition, different load balancers can make different decision on whether the server is alive or not. Accoring to our protocol, the dead servers are reported to the monitor. The question is, what to do with those balancers that consider the server is alive? Should the monitor send a delete request to those balancers? Another problem is, is the server really dead? Should we always clean up the reported servers or we waits until we have a concensus from a quorum of servers? 

As we see in the example, letting all load balancers owns the same server set is not a good idea as it leads to multiple consistent issues. Although it works if the monitor regular perform some synchronization among servers, it is ugly and cause unnecessary complexity. 

We address this issue by paritioning servers into several "zones". The concept of zone is simply a logical concept (see [Future Works](#future-works) for more details). Each load balancer is assigned a zone, which is a one-to-one mapping. A load balancer is only responsible for its zone. It only health checks the servers in the zone and report the dead servers in the zone to the monitor.

On the other hand, we assume the monitor will not have high loads so we only adopt the primary-backup setting for fault tolerance. An additional database (Redis) is also used to persist servers address. A new monitor address will be sent to all load balancers after a new leader is elected.

In this setting, we also safely handle several fault tolerance issues. 
- If an application crash, only one load balancer can detect and report it to the monitor.
- If a load balancer crash, the primary monitor can restart a load balancer and populate the servers in the zone in the initialization phase.
- If a primary monitor dead, a backup monitor will become a new leader after the election process. After the new leader is elected, the new primary monitor will be sent to all load balancers.
- It doesn't matter if a load balancer has inconsistent data with the monitor because the load balancer can still detect dead servers after recovery. The data in the monitor database is the ground-truth of the system state. As all of our updates can be made idempotent (ignore already deleted servers), it is ok for some of the dead servers pretent to be alive in the database since it will eventually be removed.


## Implementation

### Load Balancer

Internally, servers are divided into 3 pools: alive, disconnected, and dead. Only alive servers are used to serve user requests.

In the beginning, servers from the monitor will be first put into the alive pool. The alive pool is periodically checked by a background goroutine every 10 seconds. If the health check of a server fails, it would be moved into the disconnected pool.

The disconnected pool is also periodically checked every 30 seconds by another goroutine. If a server get reconnected again, it would be put back to the alive pool. Otherwise, it stays. If we retry three times, the server would be put into the dead pool and would never be checked again.

The dead pool is reported every 60 seconds by yet another goroutine. If the report is successful, the dead servers are completely deleted.


**Note**: In current implementation, we only model completely crashed server. If a server can return 500, it is not determined dead because it could probably a temporary error. To ensure this level of fault tolerance, the client library should do its own retry until it picks a health server.

**Note**: We internally use channels to synchronize the core data structure so that multiple front-end and back-end goroutines can get consistent result. The load balancer is designed to serve 10000 concurrent users. For the whole system consistency issues, please refer to [Consistency Model](#consistency-model). 


### Application

TODO

### Monitor

TODO

## API Reference

### Load Balancer
- `GET /health_check`: Return ok if the load balancer is alive.
- `GET /servers`: Return alive servers' address.
- `PUT /servers`: Add new servers and/or delete old servers.
- `PUT /monitor`: Update primary monitor address.

### Application
- `GET /health_check`: Return ok if the application is alive.

### Monitor
- `GET /containers`: Returns a JSON list of all running container IDs.
- `PUT /report`: Update dead servers reported from a load balancer.

## Future Works

TODO