package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
)
type Node struct {
	ID int
    onElection bool
	isLeader bool
}

var (
    nodes            map[int]*Node
    node             *Node
    leaderChangeChan chan struct{}

)


func main() {
	fmt.Printf("success run:\n")
	monitorID, err := strconv.Atoi(os.Getenv("monitor_ID"))
	//print monitor ID
	fmt.Printf("Monitor ID: %v\n", monitorID)
    if err != nil {
        fmt.Printf("Error converting monitor ID to int: %v", err)
    }
    nodes = map[int]*Node{
        1: {ID: 1, isLeader: false,  onElection: false},
        2: {ID: 2, isLeader: false, onElection: false},
        3: {ID: 3, isLeader: false, onElection: false},
    }

    node  = nodes[monitorID]
	
	// Create a channel to signal when the leader changes
	cli, redisClient, err := initClients()
	if err != nil {
		log.Fatalf("Error initializing clients: %v", err)
	}
	go monitorLeaderChanges(leaderChangeChan)

    go startHealthCheck(leaderChangeChan,redisClient)

	

	http.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, "alive")
    })
    http.HandleFunc("/leader", handleLeaderChange)
    http.HandleFunc("/election", handleElectionRequest)


     // Start the HTTP server
	 log.Fatal(http.ListenAndServe(":8080", nil))

	//init clients

	go manageContainers(cli)
	startHTTPServer(cli) 
	}


	
func initClients() (*client.Client, *redis.Client, error) {
	// Create a new Docker client
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
