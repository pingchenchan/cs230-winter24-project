package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"

)

func storeLeaderInRedis(leaderID int, redisClient *redis.Client) {
	// Store the leader in Redis
	redisClient.Set(context.Background(), "leader", leaderID, 0)
	//print
	fmt.Printf("Leader stored in Redis: %v\n", leaderID)
}