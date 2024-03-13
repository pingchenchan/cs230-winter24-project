package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	// "time"
	"crypto/rand"
	"encoding/hex"
	"strconv"

	"github.com/docker/go-connections/nat"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
)
func randomHex(n int) (string, error) {
    bytes := make([]byte, n)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

func manageContainers(cli *client.Client, redisClient *redis.Client) {
	fmt.Printf("manageContainers success run:\n")
	ctx := context.Background()
	// start a new container every 10 seconds

	go func() {
		// for range time.Tick(10 * time.Second) {
			fmt.Printf("Starting container\n")
			startContainer(ctx, cli, redisClient)
		// }
	}()
//wait 10 sec
	time.Sleep(10 * time.Second)

	
	// // stop the oldest container every 5 seconds
	go func() {
		// for range time.Tick(5 * time.Second) {
			stopOldestContainer(ctx, cli, redisClient)
		// }
	}()
}

func startContainer(ctx context.Context, cli *client.Client, redisClient *redis.Client) {
	portNumber := "500" + strconv.Itoa(node.ID+3)
	fmt.Printf("port: %v\n", portNumber )
	hostConfig := &container.HostConfig{
		NetworkMode: "cs230_default",
		PortBindings: nat.PortMap{
            "5001/tcp": []nat.PortBinding{
                {
                    HostIP:   "0.0.0.0",
                    HostPort: portNumber  ,
                },
            },
        },
	}
	randomString, _ := randomHex(5)
	containerConfig := &container.Config{
		Image: "cs230-flask_app"+strconv.Itoa(node.ID),
		ExposedPorts: nat.PortSet{
			"5001/tcp": struct{}{},
		},
		Env: []string{
            "KAFKA_BROKER=kafka:9092",
            "KAFKA_TOPIC=zone_1",
        },
        Cmd: []string{
            "/bin/sh", "-c",
            "wget https://github.com/jwilder/dockerize/releases/download/v0.6.1/dockerize-linux-amd64-v0.6.1.tar.gz && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-v0.6.1.tar.gz && rm dockerize-linux-amd64-v0.6.1.tar.gz && dockerize -wait tcp://kafka:9092 -timeout 30s && python app.py",
        },
	}
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "flask_app"+strconv.Itoa(node.ID) +"_"+randomString,)
	if err != nil {
		log.Printf("Error creating container: %v", err)
		return
	}

	// Store the container ID in Redis
	if err := redisClient.LPush(ctx, "containers", resp.ID).Err(); err != nil {
		log.Printf("Error storing container ID in Redis: %v", err)
		return
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Printf("Error starting container: %v", err)
		return
	}
	log.Printf("Started container: %v", resp.ID)
}

func containersHandler(w http.ResponseWriter, r *http.Request, cli *client.Client) {
	if !node.isLeader {
        http.Error(w, "Node is not the leader", http.StatusForbidden)
        return
    }
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	containerIDs, err := getRunningContainers(cli, r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(containerIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func getRunningContainers(cli *client.Client, ctx context.Context) ([]string, error) {
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	var containerIDs []string
	for _, container := range containers {
		containerIDs = append(containerIDs, container.ID)
	}
	return containerIDs, nil
}

func stopOldestContainer(ctx context.Context, cli *client.Client, redisClient *redis.Client) {
	id, err := redisClient.RPop(ctx, "containers").Result()
	if err != nil {
		if err == redis.Nil {
			log.Println("No containers to stop")
		} else {
			log.Printf("Error retrieving container ID: %v", err)
		}
		return
	}

	if err := cli.ContainerStop(ctx, id, container.StopOptions{}); err != nil {
		log.Printf("Error stopping container: %v, %v", id, err)
		return
	}
	log.Printf("Stopped container: %v", id)
}