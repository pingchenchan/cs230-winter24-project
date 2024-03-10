package gateway

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Policy interface {
	Select(*ServerPool) *Backend
}

type RoundRobinPolicy struct {
	current int
}

func (rr *RoundRobinPolicy) Select(p *ServerPool) *Backend {
	backend := p.Get(rr.current % p.Len())
	rr.current = (rr.current + 1) % p.Len()
	return backend
}

type RandomPolicy struct {
}

func (rr *RandomPolicy) Select(p *ServerPool) *Backend {
	return p.alives[rand.Intn(len(p.alives))]
}

func genLoadBalancer(gtw *Gateway) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// fmt.Println("lb")
		attempts := GetAttemptsFromContext(r)
		if attempts > 3 {
			log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
			http.Error(w, "Service not available", http.StatusServiceUnavailable)
			return
		}

		next := gtw.NextServer()
		if next != nil {
			log.Println(next)
			next.ReverseProxy.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
	}
}

func genRetryErrorHandler(serverUrl *url.URL, proxy *httputil.ReverseProxy, gtw *Gateway) func(w http.ResponseWriter, r *http.Request, err error) {
	lb := genLoadBalancer(gtw)

	return func(w http.ResponseWriter, r *http.Request, err error) {
		// fmt.Println("proxy error handler")
		// log.Printf("[%s] %s\n", serverUrl.Host, err.Error())
		retries := GetRetryFromContext(r)
		if retries < 3 {
			// fmt.Println("retry")
			<-time.After(10 * time.Millisecond)
			ctx := context.WithValue(r.Context(), Retries, retries+1)
			// fmt.Println("retry")
			proxy.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// fmt.Println("endpoint failed")

		// this server fails, remove it from our list
		gtw.Remove(serverUrl.String())

		attempts := GetAttemptsFromContext(r)
		log.Printf("%s(%s) Attempting retry %d\n", r.RemoteAddr, r.URL.Path, attempts)
		ctx := context.WithValue(r.Context(), Attempts, attempts+1)
		lb(w, r.WithContext(ctx))
	}
}
