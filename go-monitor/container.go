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
	"bytes"

	"github.com/docker/go-connections/nat"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
	"github.com/influxdata/influxdb-client-go/v2"
)
type MonitorReport struct {
    Dead []string `json:"dead"`
}
type Monitor struct {
	Url string `json:"url"`
}

type Servers struct {
	Adds    []string `json:"adds"`
	Deletes []string `json:"deletes"`
}
func randomHex(n int) (string, error) {
    bytes := make([]byte, n)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

func manageContainers(cli *client.Client, redisClient *redis.Client, influxClient influxdb2.Client) {
	fmt.Printf("manageContainers success run:\n")
	ctx := context.Background()
	zones:= []string{"zone1", "zone2", "zone3"}
	// Store the container ID in Redis
	appIDs := []string{"cs230-flask_app1-1", "cs230-flask_app2-1", "cs230-flask_app3-1"}
	appPorts := []string{"5001", "5002", "5003"}
	// time.Sleep(5 * time.Second)
	if (node.isLeader ){
		for i := 1; i <= 3; i++ {
			err := sendMonitorUpdate("http://cs230-go-monitor-"+strconv.Itoa(3)+"-1:8080", "http://go-loadbalancer-"+strconv.Itoa(i)+":5000/monitor")
			if err != nil {
				log.Fatalf("Failed to send monitor update: %v", err)
			}
		}
		log.Printf("Sent PUT /monitor update to all load balancers")
	//register 3 apps across different zones
	redisClient.FlushAll(ctx)
	for idx, appID := range appIDs {
		if err := redisClient.SAdd(ctx, "zone"+strconv.Itoa(idx+1), appID).Err(); err != nil {
			log.Printf("Error storing container ID in Redis: %v", err)
			return
		}
		//add used port(keypair format, appID,port) to redis
		if err := redisClient.Set(ctx, appID, appPorts[idx], 0).Err(); err != nil {
			log.Printf("Error storing container ID in Redis: %v", err)
			return
		}
		if err := redisClient.SAdd(ctx, "ports", appPorts[idx]).Err(); err != nil {
			log.Printf("Error storing port in Redis: %v", err)
			return
		}
		//add app to load balancer
		if err := addAppToLB("http://go-loadbalancer-"+strconv.Itoa(idx+1)+":5000", "http://"+appID+":5001"); err != nil {
			log.Fatalf("Failed to delete app from load balancer: %v", err)
		}
	
	}
	log.Printf("Registered 3 apps across different zones and added to load balancer")
	}
	

	


	

	go func() {
		for range time.Tick(10 * time.Second) {
			if (node.isLeader ){


				//print everything in redis
				for _, zone := range zones {
					containers, err := redisClient.SMembers(ctx, zone).Result()
					if err != nil {
						log.Printf("Error retrieving container IDs from Redis: %v", err)
						return
					}

					containersNumber := len(containers)
					meanCPUUsage, err := QueryData(zone, influxClient)
					if err != nil {
						log.Printf("Error querying data: %s", err)
					}
					fmt.Printf("%v running apps in %v: %v. Last 10 sec avg CPU usage is %v\n",containersNumber , zone, containers, meanCPUUsage)

					//start or stop containers based on CPU usage
					if (meanCPUUsage!=-1){
						//start a container if the CPU usage is high
						if (containersNumber<MAX_APPS_PER_ZONE && meanCPUUsage>=MAX_CPU_USAGE){

							startContainer(ctx, cli, redisClient,zone)


						}
						//stop a container if the CPU usage is low
						if (containersNumber>=2 && meanCPUUsage<=MIN_CPU_USAGE){


							//get one container from the zone
							containerName, err := redisClient.SPop(ctx, zone).Result()
							fmt.Printf("stop container id: %s in %v\n", containerName, zone)
							if err != nil {
								if err == redis.Nil {
									log.Println("No containers to stop")
								} else {
									log.Printf("Error retrieving container ID: %v", err)
								}
								return
							}

							err = delContainer(ctx, cli, redisClient, containerName,zone)
							if err != nil {
								log.Printf("Error stopping container: %v, %v", containerName , err)
								return
							}



	
						}

					}

				}

	



			}
			
		}
			
	}()




	time.Sleep(10 * time.Second)
	go func() {
		if (node.isLeader ){





		
		}
	}()
}


func delContainer(ctx context.Context, cli *client.Client, redisClient *redis.Client, containerName string,zone string) error {
	//stop the container
	if err := cli.ContainerStop(ctx, containerName , container.StopOptions{}); err != nil {
		log.Printf("Error stopping container: %v, %v", containerName , err)
		return err
	}
	//remove the container
	if err := cli.ContainerRemove(ctx, containerName, types.ContainerRemoveOptions{}); err != nil {
		log.Printf("Error removing container: %v, %v", containerName, err)
		return err
	}
	//remove the port from redis
	if err := redisClient.Del(ctx, containerName).Err(); err != nil {
		log.Printf("Error removing container ID from Redis: %v", err)
		return err
	}
	//remove the port from redis
	if err := redisClient.SRem(ctx, "ports", containerName).Err(); err != nil {
		log.Printf("Error removing port from Redis: %v", err)
		return err
	}

	zoneNumberChar := zone[len(zone)-1:]
	if err := deleteAppFromLB("http://go-loadbalancer-"+zoneNumberChar+":5000", "http://"+containerName+":5001"); err != nil {
		log.Fatalf("Failed to delete app from load balancer: %v", err)
	}
	fmt.Printf("update delete app to load balancer: %s in %v\n", containerName, zone)

	return nil
}

func reportHandler(w http.ResponseWriter, r *http.Request, cli *client.Client, redisClient *redis.Client) {
    if r.Method != http.MethodPut {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }

    var report MonitorReport
    err := json.NewDecoder(r.Body).Decode(&report)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    for _, serverName := range report.Dead {
        //search for the server in redis from all zones
        zones:= []string{"zone1", "zone2", "zone3"}
        found := false
        for _, zone := range zones {
            containers, err := redisClient.SMembers(r.Context(), zone).Result()
            if err != nil {
                http.Error(w, fmt.Sprintf("Error retrieving container IDs from Redis: %v", err), http.StatusInternalServerError)
                return
            }
            for _, containerName := range containers {
                if containerName == serverName {
                    //delete the container from redis set
                    if err := redisClient.SRem(r.Context(), zone, serverName).Err(); err != nil {
                        http.Error(w, fmt.Sprintf("Error removing container ID from Redis: %v", err), http.StatusInternalServerError)
                        return
                    }
                    
                    // Stop the server
                    if err := delContainer(r.Context(), cli, redisClient, serverName, zone); err != nil {
                        http.Error(w, err.Error(), http.StatusInternalServerError)
                        return
                    }

                    found = true
                    break
                }
            }
            if found {
                break
            }
        }
    }

    // Respond with a success message
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Servers stopped"))
}

	
	
func startContainer(ctx context.Context, cli *client.Client, redisClient *redis.Client,zone string) {
	ports, err := redisClient.SMembers(ctx, "ports").Result()
	if err != nil {
		log.Printf("Error retrieving port from Redis: %v", err)
		return
	}
	maxPort := 0
	for _, port := range ports {
		portInt, _ := strconv.Atoi(port)
		if portInt > maxPort {
			maxPort = portInt
		}
	}
	portNumber := strconv.Itoa(maxPort + 1)

	// fmt.Printf("port: %v\n", portNumber )
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
	randomString, _ := randomHex(3)
	containerConfig := &container.Config{
		Image: "cs230-flask_app"+strconv.Itoa(node.ID),
		ExposedPorts: nat.PortSet{
			"5001/tcp": struct{}{},
		},
		Env: []string{
            "KAFKA_BROKER=kafka:9092",
            "KAFKA_TOPIC="+zone,
        },
        Cmd: []string{
            "/bin/sh", "-c",
            "wget https://github.com/jwilder/dockerize/releases/download/v0.6.1/dockerize-linux-amd64-v0.6.1.tar.gz && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-v0.6.1.tar.gz && rm dockerize-linux-amd64-v0.6.1.tar.gz && dockerize -wait tcp://kafka:9092 -timeout 30s && python app.py",
        },
	}
	containerName := "flask_app"+strconv.Itoa(node.ID) +"_"+randomString
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName ,)
	if err != nil {
		log.Printf("Error creating container: %v", err)
		return
	}

	// Store the container ID in Redis
	if err := redisClient.SAdd(ctx, zone, containerName ).Err(); err != nil {
		log.Printf("Error storing container ID in Redis: %v", err)
		return
	}
	if err := redisClient.Set(ctx, containerName, portNumber, 0).Err(); err != nil {
		log.Printf("Error storing container ID in Redis: %v", err)
		return
	}
	if err := redisClient.SAdd(ctx, "ports", portNumber).Err(); err != nil {
		log.Printf("Error storing port in Redis: %v", err)
		return
	}


	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Printf("Error starting container: %v", err)
		return
	}
	log.Printf("Started container: %v", resp.ID)

	zoneNumberChar := zone[len(zone)-1:]
	if err := addAppToLB("http://go-loadbalancer-"+zoneNumberChar+":5000", "http://"+containerName+":5001"); err != nil {
		log.Fatalf("Failed to delete app from load balancer: %v", err)
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

func sendMonitorUpdate(monitorName string, url string) error {
    client := &http.Client{}
    monitor := Monitor{Url: monitorName}
    jsonData, err := json.Marshal(monitor)
    if err != nil {
        return err
    }

    req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("received non-OK response: %s", resp.Status)
    }

    return nil
}

func addAppToLB(LBurl string, addedAppUrl string) error {
    client := &http.Client{}
    servers := Servers{Adds: []string{addedAppUrl}}

    jsonData, err := json.Marshal(servers)
    if err != nil {
        return err
    }

    req, err := http.NewRequest(http.MethodPut, LBurl+"/servers", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("received non-OK response: %s", resp.Status)
    }
	//print the response
	fmt.Printf("addAppToLB received resp message: %v\n", resp.Status)

    return nil
}

func deleteAppFromLB(LBurl string, deletedAppUrl string) error {
    client := &http.Client{}
    servers := Servers{Deletes: []string{deletedAppUrl}}

    jsonData, err := json.Marshal(servers)
    if err != nil {
        return err
    }

    req, err := http.NewRequest(http.MethodPut, LBurl+"/servers", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("received non-OK response: %s", resp.Status)
    }

    return nil
}

