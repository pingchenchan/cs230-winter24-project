package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
)

func main() {
	// init clients
	cli, redisClient, err := initClients()
	if err != nil {
		log.Fatalf("Error initializing clients: %v", err)
	}


	go manageContainers(cli, redisClient)


	startHTTPServer(cli) 
}

func initClients() (*client.Client, *redis.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, nil, fmt.Errorf("creating Docker client: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	})

	return cli, redisClient, nil
}

func manageContainers(cli *client.Client, redisClient *redis.Client) {
	ctx := context.Background()

	// 每10秒启动一个新的nginx容器
	go func() {
		for range time.Tick(10 * time.Second) {
			startContainer(ctx, cli, redisClient)
		}
	}()

	// 每5秒停止最旧的容器
	go func() {
		for range time.Tick(5 * time.Second) {
			stopOldestContainer(ctx, cli, redisClient)
		}
	}()
}

func startContainer(ctx context.Context, cli *client.Client, redisClient *redis.Client) {
	containerConfig := &container.Config{
		Image: "nginx",
	}
	resp, err := cli.ContainerCreate(ctx, containerConfig, nil, nil, nil, "")
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

func startHTTPServer(cli *client.Client) {
	http.HandleFunc("/containers", func(w http.ResponseWriter, r *http.Request) {
		containersHandler(w, r, cli)
	})
	log.Println("Server is starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}

func containersHandler(w http.ResponseWriter, r *http.Request, cli *client.Client) {
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
