package main
import (
	"fmt"
	"net/http"
	"time"
)
func monitorLeaderChanges(leaderChangeChan chan struct{}) {
    var stopChan chan struct{}

    for {
        select {
        case <-leaderChangeChan:
            if stopChan != nil {
                close(stopChan)
            }

            if node.isLeader {
                stopChan = make(chan struct{})
                fmt.Printf("Node %d is the leader, start server\n", node.ID)
            }
        }
    }
}

func startHealthCheck(leaderChangeChan chan struct{}) {
    for {
        for _, n := range nodes {
            fmt.Printf("Node ID: %d, Is Leader: %v\n", n.ID, n.isLeader)
        }
        healthCheck(node, nodes, leaderChangeChan)
        time.Sleep(2 * time.Second)
    }
}


func healthCheck(node *Node , nodes map[int]*Node, leaderChangeChan chan struct{}) {
	if node.isLeader {
		 // Skip health check if the node is the leader
		 fmt.Printf("Skip health check if the node is the leader\n")
			return
		}

	// Check the health of the leader
	leader := findLeader()
	if leader == nil {
		startElection(node,leaderChangeChan )// No leader found, start a new election
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
			startElection(node,leaderChangeChan)
		} else {
			// Print health check result
			fmt.Printf("Health check result success: %v\n", resp.StatusCode)
			
			if resp.StatusCode != http.StatusOK {
				// Leader is not alive, start a new election
				fmt.Print("Leader is not alive, start a new election\n")
				startElection(node, leaderChangeChan)
			}
		
			resp.Body.Close()
		}
	}
// }
}
func startElection(node *Node, leaderChangeChan chan struct{}) {
node.onElection = true
// Send a request for votes to all other nodes
for _, n := range nodes {
	if n.ID > node.ID {
		client := &http.Client{
			Timeout: time.Second * 1,  // Set timeout to 5 seconds
		}
		resp, _ := client.Get(fmt.Sprintf("http://go-monitor-%d:8080/election?requestingNodeID=%d", n.ID, node.ID))
		// if err == nil && resp.StatusCode == http.StatusOK {
			if resp != nil {
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
//update Nodes
for _, n := range nodes {
	if n.ID != node.ID {
		n.isLeader = false
	}
}
leaderChangeChan <- struct{}{}
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
}
func findLeader() *Node {
for _, node := range nodes {
	if node.isLeader {
		return node
	}
}
return nil
}