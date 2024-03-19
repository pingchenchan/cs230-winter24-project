package gateway

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Servers struct {
	Adds    []string `json:"adds"`
	Deletes []string `json:"deletes"`
}

type Monitor struct {
	Url string `json:"url"`
}

func CreateServer(port int, gtw *Gateway) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(genLoadBalancer(gtw)),
	}
}

func CreateAdminServer(gtw *Gateway) *gin.Engine {
	r := gin.Default()

	r.GET("/health_check", func(c *gin.Context) {
		c.String(http.StatusOK, "alive")
	})

	r.GET("/servers", func(c *gin.Context) {
		servers := []string{}
		for _, server := range gtw.copyAlive() {
			servers = append(servers, server.rawUrl())
		}
		c.JSON(http.StatusOK, servers)
	})

	r.PUT("/servers", func(c *gin.Context) {
		var servers Servers

		if err := c.ShouldBind(&servers); err != nil {
			c.JSON(http.StatusBadRequest, nil)
			return
		}

		for _, a := range servers.Adds {
			gtw.Add(NewBackend(a, gtw))
		}

		for _, d := range servers.Deletes {
			gtw.Erase(d)
		}

		c.JSON(http.StatusOK, nil)
	})

	r.PUT("/monitor", func(c *gin.Context) {
		var monitor Monitor
		if err := c.ShouldBind(&monitor); err != nil {
			c.JSON(http.StatusBadRequest, nil)
			return
		}

		gtw.monitorUrl = monitor.Url
		c.JSON(http.StatusOK, nil)
	})
	r.GET("/monitor", func(c *gin.Context) {
		c.String(http.StatusOK, gtw.monitorUrl)
	})

	return r
}
