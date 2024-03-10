package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/health_check", func(c *gin.Context) {
		c.JSON(200, "ok")
	})

	r.GET("/id", func(c *gin.Context) {
		id := os.Getenv("APP_ID")
		c.JSON(200, fmt.Sprintf("%s:%d", id, rand.Intn(1000_000)))
	})

	if err := r.Run(":8000"); err != nil {
		log.Println(err)
	}
}
