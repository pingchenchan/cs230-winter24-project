package main
import (
	"fmt"
	"net/http"
	"time"
	"strconv"
	"github.com/redis/go-redis/v9"
	"io"
	"log"

)
func handleLeaderChange(w http.ResponseWriter, r *http.Request, leaderChangeChan chan struct{}) {
	// Check the ID of the leader
	leaderID, err := strconv.Atoi(r.URL.Query().Get("leaderID"))
	if err != nil {
		http.Error(w, "Invalid leaderID", http.StatusBadRequest)
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
	leaderChangeChan <- struct{}{} // Signal that the leader has changed
	// Respond with an empty message
	fmt.Fprint(w, "")
}
func handleElectionRequest(w http.ResponseWriter, r *http.Request) {
	// Check the ID of the requesting node
	requestingNodeID, err := strconv.Atoi(r.URL.Query().Get("requestingNodeID"))
	if err != nil {
		http.Error(w, "Invalid requestingNodeID", http.StatusBadRequest)
		return
	}

	// If the requesting node's ID is less than the current node's ID, respond with "OK"
	if requestingNodeID < node.ID {
		fmt.Fprintf(w, "OK, Leader is %d", findLeader().ID)
	} else {
		http.Error(w, "Not OK", http.StatusForbidden)
	}
}
func monitorLeaderChanges(leaderChangeChan chan struct{}) {
    var stopChan chan struct{}

    for {
        select {
        case <-leaderChangeChan:
            if stopChan != nil {
                close(stopChan)
                stopChan = nil
            }

            if node.isLeader {
                if stopChan == nil {
                    stopChan = make(chan struct{})
                }
                fmt.Printf("Node %d is the leader, start server\n", node.ID)
            }
        }
    }
}

func startHealthCheck(leaderChangeChan chan struct{},redisClient *redis.Client) {
    for {
        for _, n := range nodes {
			if n.isLeader{
				fmt.Printf("Node ID: %d, Is Leader\n", n.ID)
			}
            
        }
        healthCheck(node, nodes, leaderChangeChan,redisClient)
        time.Sleep(2 * time.Second)
    }
}


func healthCheck(node *Node , nodes map[int]*Node, leaderChangeChan chan struct{},redisClient *redis.Client) {
	if node.isLeader {
		 // Skip health check if the node is the leader
		 fmt.Printf("Skip health check if the node is the leader\n")
			return
		}

	// Check the health of the leader
	leader := findLeader()
	if leader == nil {
		fmt.Print("No leader found, start a new election\n")
		startElection(node,leaderChangeChan, redisClient )// No leader found, start a new election
	}else{
		// Check if the leader is alive
		// If the leader is not alive, start a new election
		// If the leader is alive, continue
		client := &http.Client{
			Timeout: time.Second * HEALTH_CHECK_INTERVAL,  // Set timeout to 1 second
		}
		resp, err := client.Get(fmt.Sprintf("http://go-monitor-%d:8080/alive", leader.ID))
		
		if err != nil {
			// Print the error and start a new election
			fmt.Printf("Health check error: %v\n", err)
			fmt.Print("Start a new election due to error\n")
			startElection(node,leaderChangeChan,redisClient)
		} else {
			// Print health check result
			fmt.Printf("Health check  result success: %v\n", resp.StatusCode)
			
			if resp.StatusCode != http.StatusOK {
				// Leader is not alive, start a new election
				fmt.Print("Leader is not alive, start a new election\n")
				startElection(node, leaderChangeChan, redisClient)
			}
		
			resp.Body.Close()
		}
	}
// }
}
func startElection(node *Node, leaderChangeChan chan struct{},redisClient *redis.Client) {
	node.onElection = true
	// Send a request for votes to all other nodes
	for _, n := range nodes {
		if n.ID > node.ID {
			client := &http.Client{
				Timeout: time.Second * 1,  // Set timeout to 5 seconds
			}
			resp, err := client.Get(fmt.Sprintf("http://go-monitor-%d:8080/election?requestingNodeID=%d", n.ID, node.ID))
			//print received resp message
			
			if err != nil {
				//2 cases, 1. node is not alive, 2. node ID less than current node
				// Handle error
				fmt.Printf("Error reading response body: %v\n", err)
				// node.isLeader = false
				// node.onElection = false
				// return
			}
			
			if  resp != nil && resp.StatusCode == http.StatusOK {//only higher ID node will respond with OK
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("received resp message: %v\n", string(body))
				// if resp != nil {
				// Parse the response to get the new leader ID
				var leaderID int
		
				
				fmt.Sscanf(string(body), "OK, Leader is %d", &leaderID)
				//print the leaderID
				fmt.Printf("received Leader ID: %v\n", leaderID)
				//set new leader
				for _, n := range nodes {
					if n.ID == leaderID {
						n.isLeader = true
						fmt.Printf("New leader set to: %v\n", n.ID)
					} else {
						n.isLeader = false
					}
				}
				///print election result
				fmt.Printf("Election result loser: %v\n", resp.StatusCode)
				// Received response from a higher ID node, revert to follower, and exit election
				node.isLeader = false
				node.onElection = false
				return
			}

		}
	} 
	// If no higher ID node responded, become the leader
	node.isLeader = true
	node.onElection = false
	//print election result
	fmt.Printf("Election result winner: %v\n", node.ID)
	// go storeLeaderInRedis(node.ID, redisClient)
	//update Nodes
	for _, n := range nodes {
		if n.ID != node.ID {
			n.isLeader = false
		}
	}
	leaderChangeChan <- struct{}{}
	//store leader in redis
	
	//send message to all nodes
	for _, n := range nodes {
		if n.ID != node.ID {
			client := &http.Client{
				Timeout: time.Second * 1,  // Set timeout to 5 seconds
			}
			resp, err := client.Get(fmt.Sprintf("http://go-monitor-%d:8080/leader?leaderID=%d", n.ID, node.ID))
			fmt.Printf("send message to all nodes:%v\n", n.ID)
			if err != nil || resp.StatusCode != http.StatusOK {
				fmt.Printf("Error sending message to node: %v\n", err)
			}
		}
	}

	for i := 1; i <= 3; i++ {
		err := sendMonitorUpdate("go-monitor-"+strconv.Itoa(node.ID), "http://go-loadbalancer-"+strconv.Itoa(i)+":5000/monitor")
		if err != nil {
			log.Fatalf("Failed to send monitor update: %v", err)
		}
	}
	log.Printf("Sent monitor update to all load balancers")
}
func findLeader() *Node {
	for _, node := range nodes {
		if node.isLeader {
			return node
		}
	}
	return nil
}