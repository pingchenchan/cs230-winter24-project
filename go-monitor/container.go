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
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
	"github.com/influxdata/influxdb-client-go/v2"
)
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

	//clear redis
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
	
	}


	
	//register 3 apps across different zones
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

					if (meanCPUUsage!=-1){
						if (containersNumber<MAX_APPS_PER_ZONE && meanCPUUsage>=MAX_CPU_USAGE){

							startContainer(ctx, cli, redisClient,zone)
						}
						if (containersNumber>=2 && meanCPUUsage<=MIN_CPU_USAGE){


							//get one container from the zone
							containerName , err := redisClient.SPop(ctx, zone).Result()
							fmt.Printf("stop container id: %s in %v\n", containerName,zone)
							if err != nil {
								if err == redis.Nil {
									log.Println("No containers to stop")
								} else {
									log.Printf("Error retrieving container ID: %v", err)
								}
								return
							}
							//get the maping port from the container

							//stop the container
							if err := cli.ContainerStop(ctx, containerName , container.StopOptions{}); err != nil {
								log.Printf("Error stopping container: %v, %v", containerName , err)
								return
							}
							//remove the container
							if err := cli.ContainerRemove(ctx, containerName, types.ContainerRemoveOptions{}); err != nil {
								log.Printf("Error removing container: %v, %v", containerName, err)
								return
							}
							//remove the port from redis
							if err := redisClient.Del(ctx, containerName).Err(); err != nil {
								log.Printf("Error removing container ID from Redis: %v", err)
								return
							}
							//remove the port from redis
							if err := redisClient.SRem(ctx, "ports", containerName).Err(); err != nil {
								log.Printf("Error removing port from Redis: %v", err)
								return
							}

	
						}

//print used port in redis
						// ports, err := redisClient.SMembers(ctx, "ports").Result()
						// if err != nil {
						// 	log.Printf("Error retrieving port from Redis: %v", err)
						// 	return
						// }
						// fmt.Printf("ports: %v\n", ports)
					}
	
						// startContainer(ctx, cli, redisClient,zone)
					
					

				}

	



			}
			
		}
			
	}()




	time.Sleep(10 * time.Second)
	go func() {
		if (node.isLeader ){





		//stop all containers in zone 1


		// ...

		// containers, err := redisClient.SMembers(ctx, "zone1").Result()
		// if err != nil {
		// 	log.Printf("Error retrieving container IDs from Redis: %v", err)
		// 	return
		// }
		// for _, id := range containers {
			// id := "cs230-flask_app1-1"
			// if err := cli.ContainerStop(ctx, id, container.StopOptions{}); err != nil {
			// 	log.Printf("Error stopping container: %v, %v", id, err)
			// 	return
			// }
			// log.Printf("Stopped container: %v", id)
		// }


		// for range time.Tick(5 * time.Second) {
			// stopOldestContainer(ctx, cli, redisClient)
		// }
		}
	}()
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