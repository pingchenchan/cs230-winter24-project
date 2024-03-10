package main
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
)
func manageContainers(cli *client.Client) {
	// ctx := context.Background()
	//start a new container every 10 seconds
	// go func() {
	// 	for range time.Tick(10 * time.Second) {
	// 		startContainer(ctx, cli, redisClient)
	// 	}
	// }()

	// // stop the oldest container every 5 seconds
	// go func() {
	// 	for range time.Tick(5 * time.Second) {
	// 		stopOldestContainer(ctx, cli, redisClient)
	// 	}
	// }()
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