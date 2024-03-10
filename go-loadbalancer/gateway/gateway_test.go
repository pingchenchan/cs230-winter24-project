package gateway

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/emirpasic/gods/sets/hashset"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

type closeNotifyingRecorder struct {
	*httptest.ResponseRecorder
	closeNotifyChan chan bool
}

func newCloseNotifyingRecorder() *closeNotifyingRecorder {
	return &closeNotifyingRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
}

func (cnr *closeNotifyingRecorder) CloseNotify() <-chan bool {
	return cnr.closeNotifyChan
}

func TestGatewayCoreSequential(t *testing.T) {
	gtw := NewGateway()

	assert.Equal(t, 0, len(gtw.serverPool.alives))
	assert.Equal(t, 0, len(gtw.serverPool.disconnected))

	serverNum := 10000
	backends := []*Backend{}
	for i := 0; i < serverNum; i++ {
		backend := NewBackend(fmt.Sprintf("http://localhost:%d", 8000+i), gtw)
		backends = append(backends, backend)
		gtw.Add(backend)
	}

	assert.Equal(t, serverNum, len(gtw.serverPool.alives))
	assert.Equal(t, 0, len(gtw.serverPool.disconnected))

	for i := 0; i < serverNum; i++ {
		assert.Equal(t, backends[i].rawUrl(), gtw.serverPool.alives[i].rawUrl())
	}

	removedBackendsIds := map[int]bool{10: true, 27: true, 43: true}
	for id := range removedBackendsIds {
		gtw.Remove(backends[id].rawUrl())
	}

	assert.Equal(t, serverNum-len(removedBackendsIds), len(gtw.serverPool.alives))
	assert.Equal(t, len(removedBackendsIds), len(gtw.serverPool.disconnected))

	next := 0
	for i, b := range backends {
		if _, ok := removedBackendsIds[i]; ok {
			continue
		}
		assert.Equal(t, b.rawUrl(), gtw.serverPool.alives[next].rawUrl())
		next++
	}

	// test repeated selection and remove
	crashedServerIds := []int{80, 88, 97}
	for _, id := range crashedServerIds {
		gtw.Remove(backends[id].rawUrl())
		removedBackendsIds[id] = true
		for i, b := range backends {
			if _, ok := removedBackendsIds[i]; ok {
				continue
			}
			backend := gtw.NextServer()
			assert.Equal(t, b.rawUrl(), backend.rawUrl())
		}
	}
}

func TestGatewayCoreConcurrent(t *testing.T) {
	gtw := NewGateway()

	assert.Equal(t, 0, len(gtw.serverPool.alives))
	assert.Equal(t, 0, len(gtw.serverPool.disconnected))

	serverNum := 10000
	backendSet := hashset.New()
	for i := 0; i < serverNum; i++ {
		backendSet.Add(fmt.Sprintf("http://localhost:%d", 8000+i))
	}

	var wg sync.WaitGroup
	wg.Add(serverNum)
	for _, rawUrl := range backendSet.Values() {
		go func(rawUrl string) {
			backend := NewBackend(rawUrl, gtw)
			gtw.Add(backend)
			wg.Done()
		}(rawUrl.(string))
	}
	wg.Wait()

	assert.Equal(t, serverNum, backendSet.Size())
	assert.Equal(t, serverNum, len(gtw.serverPool.alives))
	assert.Equal(t, 0, len(gtw.serverPool.disconnected))

	servers := gtw.copyAlive()
	for i := 0; i < serverNum; i++ {
		assert.True(t, backendSet.Contains(servers[i].rawUrl()))
	}

	removedNum := 5000
	removedSet := hashset.New()
	for i := 0; i < removedNum; i++ {
		removedSet.Add(fmt.Sprintf("http://localhost:%d", 8000+i))
	}

	wg.Add(removedNum)
	for _, rawUrl := range removedSet.Values() {
		go func(rawUrl string) {
			gtw.Remove(rawUrl)
			wg.Done()
		}(rawUrl.(string))
	}
	wg.Wait()

	assert.Equal(t, serverNum-removedNum, len(gtw.serverPool.alives))
	assert.Equal(t, removedNum, len(gtw.serverPool.disconnected))

	remainingServers := gtw.copyAlive()
	remainingSet := backendSet.Difference(removedSet)
	for i := 0; i < len(remainingServers); i++ {
		rawUrl := remainingServers[i].rawUrl()
		assert.True(t, remainingSet.Contains(rawUrl))
		assert.False(t, removedSet.Contains(rawUrl))
	}

	// test repeated selection and remove
	newRemovedNum := 100
	newRemovedServers := hashset.New()
	for i := 0; i < newRemovedNum; i++ {
		newRemovedServers.Add(fmt.Sprintf("http://localhost:%d", 8000+removedNum+10*i))
	}

	for _, v := range newRemovedServers.Values() {
		rawUrl := v.(string)
		gtw.Remove(rawUrl)
		remainingSet.Remove(rawUrl)
		slen := gtw.serverPool.Len()

		assert.Equal(t, remainingSet.Size(), slen)
		resultSet := hashset.New()
		resultCh := make(chan string)
		doneCh := make(chan struct{})
		wg.Add(slen)
		for i := 0; i < slen; i++ {
			go func() {
				resultCh <- gtw.NextServer().rawUrl()
				wg.Done()
			}()
		}

		go func() {
			for i := 0; i < slen; i++ {
				resultSet.Add(<-resultCh)
			}
			doneCh <- struct{}{}
		}()

		<-doneCh
		wg.Wait()

		// all selections should contains the entire sets of remaing servers
		assert.Equal(t, remainingSet.Size(), resultSet.Size())
		diff := remainingSet.Difference(resultSet)
		assert.True(t, diff.Empty())
	}
}

func TestGatewayHealthCheckReport(t *testing.T) {
	gtw := NewGateway()

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	serverNum := 10000
	for i := 0; i < serverNum; i++ {
		backend := NewBackend(fmt.Sprintf("http://localhost:%d", 8000+i), gtw)
		gtw.Add(backend)
	}

	aliveSet := hashset.New()
	disconnectedSet := hashset.New()
	for i := 0; i < serverNum; i++ {
		rawUrl := fmt.Sprintf("http://localhost:%d", 8000+i)
		if i%100 == 0 {
			httpmock.RegisterResponder("GET", rawUrl+"/health_check",
				httpmock.NewStringResponder(500, fmt.Sprintf("Service %d is Down", i)))
			disconnectedSet.Add(rawUrl)
		} else {
			httpmock.RegisterResponder("GET", rawUrl+"/health_check",
				httpmock.NewStringResponder(200, fmt.Sprintf("Service %d is Alive", i)))
			aliveSet.Add(rawUrl)
		}
	}

	// register monitor endpoint
	monitorUrl := "http://localhost:3000/report"
	httpmock.RegisterResponder("POST", monitorUrl, httpmock.NewStringResponder(200, "Success"))

	assert.Equal(t, serverNum, len(gtw.serverPool.alives))
	assert.Equal(t, 0, len(gtw.serverPool.disconnected))
	assert.Equal(t, 0, len(gtw.serverPool.dead))

	// do health check
	duration := time.Millisecond
	healthCheckWithCount(gtw, duration, 1)

	assert.Equal(t, aliveSet.Size(), len(gtw.serverPool.alives))
	assert.Equal(t, disconnectedSet.Size(), len(gtw.serverPool.disconnected))

	remainingSet := hashset.New()
	remainingServers := gtw.copyAlive()
	for _, s := range remainingServers {
		remainingSet.Add(s.rawUrl())
	}

	diff := remainingSet.Difference(aliveSet)
	assert.True(t, diff.Empty())

	// Resurrect some disconnected servers
	resurrectCount := 0
	for i := 0; i < serverNum; i++ {
		rawUrl := fmt.Sprintf("http://localhost:%d", 8000+i)
		if i%200 == 0 {
			httpmock.RegisterResponder("GET", rawUrl+"/health_check",
				httpmock.NewStringResponder(200, fmt.Sprintf("Service %d is Alive", i)))
			aliveSet.Add(rawUrl)
			disconnectedSet.Remove(rawUrl)
			resurrectCount++
		}
	}
	gtw.resurrectDisconnected()
	assert.Equal(t, aliveSet.Size(), len(gtw.serverPool.alives))
	assert.Equal(t, disconnectedSet.Size(), len(gtw.serverPool.disconnected))
	assert.Equal(t, 0, len(gtw.serverPool.dead))

	// Retry resurrect several times
	gtw.resurrectDisconnected()
	gtw.resurrectDisconnected()
	deadSet := disconnectedSet.Difference(hashset.New())
	disconnectedSet.Clear()
	assert.Equal(t, aliveSet.Size(), len(gtw.serverPool.alives))
	assert.Equal(t, disconnectedSet.Size(), len(gtw.serverPool.disconnected))
	assert.Equal(t, deadSet.Size(), len(gtw.serverPool.dead))
}

func TestGatewayHTTP(t *testing.T) {
	gtw := NewGateway()

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:8000/",
		httpmock.NewStringResponder(200, "Service 0"))

	gtw.Add(NewBackend("http://localhost:8000", gtw))
	gtw.Add(NewBackend("http://localhost:8001", gtw))

	assert.Equal(t, 2, len(gtw.serverPool.alives))
	assert.Equal(t, 0, len(gtw.serverPool.disconnected))
	assert.Equal(t, 0, len(gtw.serverPool.dead))

	req, _ := http.NewRequest("GET", "/", nil)
	resp := newCloseNotifyingRecorder()

	server := CreateServer(3000, gtw)

	server.Handler.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "Service 0", resp.Body.String())

	// service 1 fails -> connect to service 0
	req, _ = http.NewRequest("GET", "/", nil)
	resp = newCloseNotifyingRecorder()
	server.Handler.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "Service 0", resp.Body.String())

	// service one comes back
	httpmock.RegisterResponder("GET", "http://localhost:8001/",
		httpmock.NewStringResponder(200, "Service 1"))
	httpmock.RegisterResponder("GET", "http://localhost:8001/health_check",
		httpmock.NewStringResponder(200, "Service 1"))

	gtw.resurrectDisconnected()
	assert.Equal(t, 2, len(gtw.serverPool.alives))
	assert.Equal(t, 0, len(gtw.serverPool.disconnected))
	assert.Equal(t, 0, len(gtw.serverPool.dead))

	req, _ = http.NewRequest("GET", "/", nil)
	resp = newCloseNotifyingRecorder()
	server.Handler.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "Service 0", resp.Body.String())

	assert.Equal(t, 2, len(gtw.serverPool.alives))
	assert.Equal(t, 0, len(gtw.serverPool.disconnected))
	assert.Equal(t, 0, len(gtw.serverPool.dead))

	req, _ = http.NewRequest("GET", "/", nil)
	resp = newCloseNotifyingRecorder()
	server.Handler.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "Service 1", resp.Body.String())
}

func TestGatewayMonitor(t *testing.T) {
	gtw := NewGateway()

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	gtw.monitorUrl = "http://localhost:6000"

	httpmock.RegisterResponder("PUT", "http://localhost:6000",
		httpmock.NewStringResponder(200, "Service 0"))

	serverNum := 100
	servers := []string{}
	for i := 0; i < serverNum; i++ {
		backend := NewBackend(fmt.Sprintf("http://localhost:%d", 8000+i), gtw)
		gtw.Add(backend)
		servers = append(servers, backend.rawUrl())
	}

	for i := 0; i < len(servers); i++ {
		if i%2 == 0 {
			gtw.Erase(servers[i])
		} else if i%5 == 0 {
			gtw.Remove(servers[i])
			if i%15 == 0 {
				gtw.Disconnect(servers[i])
			}
		}
	}

	assert.Equal(t, 40, len(gtw.serverPool.alives))
	assert.Equal(t, 7, len(gtw.serverPool.disconnected))
	assert.Equal(t, 3, len(gtw.serverPool.dead))

	err := gtw.report()
	assert.Nil(t, err)
	assert.Equal(t, 40, len(gtw.serverPool.alives))
	assert.Equal(t, 7, len(gtw.serverPool.disconnected))
	assert.Equal(t, 0, len(gtw.serverPool.dead))
}

// func TestCreateRouter(t *testing.T) {
// 	// Activate the httpmock library
// 	httpmock.Activate()
// 	defer httpmock.DeactivateAndReset()

// 	// Mock the servers
// 	httpmock.RegisterResponder("GET", "http://localhost:8081/test",
// 		httpmock.NewStringResponder(200, "Service 1"))
// 	httpmock.RegisterResponder("GET", "http://localhost:8082/test",
// 		httpmock.NewStringResponder(200, "Service 2"))

// 	// Create the router
// 	router := CreateRouter()

// 	// Create a request to the first route
// 	req, _ := http.NewRequest("GET", "/service1/test", nil)
// 	resp := newCloseNotifyingRecorder()

// 	// Serve the request
// 	router.ServeHTTP(resp, req)

// 	// Check the status code
// 	assert.Equal(t, http.StatusOK, resp.Code)
// 	assert.Equal(t, "Service 1", resp.Body.String())

// 	// Create a request to the second route
// 	req, _ = http.NewRequest("GET", "/service2/test", nil)
// 	resp = newCloseNotifyingRecorder()

// 	// Serve the request
// 	router.ServeHTTP(resp, req)

// 	// Check the status code
// 	assert.Equal(t, http.StatusOK, resp.Code)
// 	assert.Equal(t, "Service 2", resp.Body.String())
// }
