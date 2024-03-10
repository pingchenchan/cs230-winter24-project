package gateway

// func CreateRouter() *gin.Engine {
// 	router := gin.Default()

// 	// Define the routes for the API Gateway
// 	router.Any("/service1/*path", createProxy("http://localhost:8081"))
// 	router.Any("/service2/*path", createProxy("http://localhost:8082"))

// 	return router
// }

// func StartRouter(router *gin.Engine) {
// 	// Start the API Gateway
// 	router.Run(":8080")
// }

// // source 1: https://gist.github.com/seblegall/2a2697fc56417b24a7ec49eb4a8d7b1b
// // source 2: https://levelup.gitconnected.com/build-api-gateway-using-go-gin-a-comprehensive-guide-for-gateway-2-6bc1f5bd79a3

// func createProxy(target string) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		// Parse the target URL
// 		targetURL, _ := url.Parse(target)

// 		// Create the reverse proxy
// 		proxy := httputil.NewSingleHostReverseProxy(targetURL)

// 		// Modify the request
// 		// c.Request.URL.Scheme = targetURL.Scheme
// 		// c.Request.URL.Host = targetURL.Host
// 		// c.Request.URL.Path = c.Param("path")

// 		proxy.Director = func(req *http.Request) {
// 			req.Header = c.Request.Header
// 			req.Host = targetURL.Host
// 			req.URL.Scheme = targetURL.Scheme
// 			req.URL.Host = targetURL.Host
// 			req.URL.Path = c.Param("path")
// 		}

// 		// Let the reverse proxy do its job
// 		proxy.ServeHTTP(c.Writer, c.Request)
// 	}
// }
