#!/bin/bash
# Stop and remove any old containers
docker-compose down

# Start the services
docker-compose up  &

# # Wait for 10 seconds
sleep 10

# Stop the go-monitor-1 service
docker-compose stop go-monitor-3

# Remove the go-monitor-1 container
docker-compose down go-monitor-3

# # Wait for 10 seconds
sleep 10

# Stop the go-monitor-1 service
docker-compose stop go-monitor-2

# Remove the go-monitor-1 container
docker-compose down go-monitor-2

# # Wait for 10 seconds
sleep 10

# Stop the go-monitor-1 service
docker-compose stop go-monitor-1

# Remove the go-monitor-1 container
docker-compose down go-monitor-1