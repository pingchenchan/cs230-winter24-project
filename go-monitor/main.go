package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
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

var nodes = make(map[int]*Node)

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

    node, ok := nodes[monitorID]
    if !ok {
        fmt.Printf("No node found with ID: %v\n", monitorID)
        return
    }
	// Start health check
	go func() {
        for {
			for _, n := range nodes {
				fmt.Printf("Node ID: %d, Is Leader: %v\n", n.ID, n.isLeader)
			}
            healthCheck(node ,nodes)
			//print start health check
            time.Sleep(2 * time.Second)
        }
    }()
	startTime := time.Now()
	http.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
        // Respond with an empty message
		if node.ID == 3 && time.Since(startTime).Seconds() > 10 {
			fmt.Print("Killing node 3\n")
			time.Sleep(3 * time.Second) 
            return
        }
        fmt.Fprint(w, "alive")
    })

	http.HandleFunc("/leader", func(w http.ResponseWriter, r *http.Request) {
		// Check the ID of the leader
		leaderID, err := strconv.Atoi(r.URL.Query().Get("leaderID"))
		if err != nil {
			http.Error(w, "Invalid leaderID", http.StatusBadRequest)
			return
		}
		if node.ID == 3 && time.Since(startTime).Seconds() > 10 {
			fmt.Print("Killing node 3\n")
			time.Sleep(3 * time.Second) 
			
            return
        }

		// Update the leader
		for _, n := range nodes {
			 if n.ID == leaderID {
				 n.isLeader = true
			 } else {
				 n.isLeader = false
			 }
		}
		// Respond with an empty message
		fmt.Fprint(w, "")
	})
    http.HandleFunc("/election", func(w http.ResponseWriter, r *http.Request) {
		if node.ID == 3 && time.Since(startTime).Seconds() > 10 {
			fmt.Print("Killing node 3\n")
			time.Sleep(3 * time.Second) 
            return
        }
		 // Check the ID of the requesting node
		 requestingNodeID, err := strconv.Atoi(r.URL.Query().Get("requestingNodeID"))
		 if err != nil {
			 http.Error(w, "Invalid requestingNodeID", http.StatusBadRequest)
			 return
		 }
 
		 // If the requesting node's ID is less than the current node's ID, respond with "OK"
		 if requestingNodeID < node.ID {
			 fmt.Fprint(w, "OK")
		 } else {
			 http.Error(w, "Not OK", http.StatusForbidden)
		 }
		})
     // Start the HTTP server
	 http.ListenAndServe(":8080", nil)

	// init clients
	// cli, _, err := initClients()
	// if err != nil {
	// 	log.Fatalf("Error initializing clients: %v", err)
	// }
	// go manageContainers(cli, redisClient)
	// startHTTPServer(cli) 
}
func healthCheck(node *Node , nodes map[int]*Node) {
    // ticker := time.NewTicker(1 * time.Second)
    // for range ticker.C { // Run every 5 seconds
		//print health check
		// fmt.Printf("start Health check result: %v\n", node.ID)
        if node.isLeader {
			 // Skip health check if the node is the leader
			 fmt.Printf("Skip health check if the node is the leader\n")
				return
			}

        // Check the health of the leader
		leader := findLeader()
		if leader == nil {
			// No leader found, start a new election
			startElection(node)
		}else{
			// Check if the leader is alive
			// If the leader is not alive, start a new election
			// If the leader is alive, continue
			client := &http.Client{
				Timeout: time.Second * 1,  // Set timeout to 1 second
			}
			resp, err := client.Get(fmt.Sprintf("http://go-monitor-%d:8080/alive", leader.ID))
			
			if err != nil {
				// Print the error and start a new election
				fmt.Printf("Health check error: %v\n", err)
				fmt.Print("Start a new election due to error\n")
				startElection(node)
			} else {
				// Print health check result
				fmt.Printf("Health check result: %v\n", resp.StatusCode)
				
				if resp.StatusCode != http.StatusOK {
					// Leader is not alive, start a new election
					fmt.Print("Leader is not alive, start a new election\n")
					startElection(node)
				}
			
				resp.Body.Close()
			}
		}
    // }
}
func startElection(node *Node) {
	node.onElection = true
	// Send a request for votes to all other nodes
	for _, n := range nodes {
		if n.ID > node.ID {
			client := &http.Client{
				Timeout: time.Second * 1,  // Set timeout to 5 seconds
			}
			resp, err := client.Get(fmt.Sprintf("http://go-monitor-%d:8080/election?requestingNodeID=%d", n.ID, node.ID))
            // fmt.Printf("Election result 1 : %v\n", resp.StatusCode)
			if err == nil && resp.StatusCode == http.StatusOK {
				///print election result
				fmt.Printf("Election result loser: %v\n", resp.StatusCode)

                // Received response from a higher ID node, revert to follower, and exit election
                node.isLeader = false
				node.onElection = false
                return
            }else{
				fmt.Print("Election result fail: %v\n", n.ID)
			}

		}
	}
	// If no higher ID node responded, become the leader
	node.isLeader = true
	node.onElection = false
	//print election result
	fmt.Printf("Election result winner: %v\n", node.ID)
	//update Nodes
	for _, n := range nodes {
		if n.ID != node.ID {
			n.isLeader = false
		}
	}
	//send message to all nodes
	for _, n := range nodes {
		if n.ID != node.ID {
			client := &http.Client{
				Timeout: time.Second * 1,  // Set timeout to 5 seconds
			}
			resp, err := client.Get(fmt.Sprintf("http://go-monitor-%d:8080/leader?leaderID=%d", n.ID, node.ID))
			fmt.Printf("send message to all nodes: \n")
			if err != nil || resp.StatusCode != http.StatusOK {
				fmt.Printf("Error sending message to node: %v\n", err)


				// Error sending message to node
				// Log the error
			}
		}
	}



}
func findLeader() *Node {
	for _, node := range nodes {
		if node.isLeader {
			return node
		}
	}
    return nil
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
