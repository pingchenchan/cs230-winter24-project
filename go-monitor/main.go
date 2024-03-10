package main

import (

	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
        3: {ID: 3, isLeader: true, onElection: false},
    }

    node  = nodes[monitorID]
	
	// Create a channel to signal when the leader changes
	cli, redisClient, err := initClients()
	
	if err != nil {
		log.Fatalf("Error initializing clients: %v", err)
	}


	

	


    //  // Start the HTTP server
	//  log.Fatal(http.ListenAndServe(":8080", nil))

	//init clients

	go manageContainers(cli,redisClient)
	leaderChangeChan := make(chan struct{})
	
	go monitorLeaderChanges(leaderChangeChan)
    go startHealthCheck(leaderChangeChan,redisClient)
	startHTTPServer(cli, leaderChangeChan) 
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




func startHTTPServer(cli *client.Client, leaderChangeChan chan struct{}) {
	http.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, "alive")
    })
    http.HandleFunc("/leader", func(w http.ResponseWriter, r *http.Request) {
        handleLeaderChange(w, r, leaderChangeChan)
    })
    http.HandleFunc("/election", handleElectionRequest)
	http.HandleFunc("/containers", func(w http.ResponseWriter, r *http.Request) {
		containersHandler(w, r, cli)
	})
	log.Println("Server is starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}


