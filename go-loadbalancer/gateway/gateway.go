package gateway

import (
	"errors"
	"log"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

var ErrNoSuchUrl = errors.New("no such url")

var DefaultGateway = NewGateway()

func Reset() {
	DefaultGateway = NewGateway()
}

func NewBackend(rawUrl string, gtw *Gateway) *Backend {
	serverUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil
	}

	proxy := httputil.NewSingleHostReverseProxy(serverUrl)
	proxy.ErrorHandler = genRetryErrorHandler(serverUrl, proxy, gtw)

	return &Backend{
		URL:          serverUrl,
		ReverseProxy: proxy,
		Retries:      0,
	}
}

type Backend struct {
	URL          *url.URL
	ReverseProxy *httputil.ReverseProxy
	Retries      int
}

func (b *Backend) rawUrl() string {
	return b.URL.String()
}

func NewServerPool() *ServerPool {
	return &ServerPool{
		alives:       []*Backend{},
		disconnected: []*Backend{},
		dead:         []*Backend{},
	}
}

type ServerPool struct {
	alives       []*Backend
	disconnected []*Backend
	dead         []*Backend
}

func (p *ServerPool) Len() int {
	return len(p.alives)
}

func (p *ServerPool) Get(index int) *Backend {
	return p.alives[index]
}

func (p *ServerPool) Add(backends ...*Backend) {
	p.alives = append(p.alives, backends...)
}

func serverIndex(backends []*Backend, rawUrl string) (int, bool) {
	for i, b := range backends {
		if rawUrl == b.rawUrl() {
			return i, true
		}
	}
	return -1, false
}

// alive -> disconnected
func (p *ServerPool) Remove(rawUrl string) error {
	index, ok := serverIndex(p.alives, rawUrl)
	if !ok {
		return ErrNoSuchUrl
	}

	backend := p.alives[index]
	p.alives = append(p.alives[:index], p.alives[index+1:]...)
	p.disconnected = append(p.disconnected, backend)
	return nil
}

// alive -> null
func (p *ServerPool) Erase(rawUrl string) error {
	index, ok := serverIndex(p.alives, rawUrl)
	if !ok {
		return ErrNoSuchUrl
	}

	p.alives = append(p.alives[:index], p.alives[index+1:]...)
	return nil
}

// disconnected -> dead
func (p *ServerPool) Disconnect(rawUrl string, delete bool) (*Backend, error) {
	index, ok := serverIndex(p.disconnected, rawUrl)
	if !ok {
		return nil, ErrNoSuchUrl
	}
	backend := p.disconnected[index]
	p.disconnected = append(p.disconnected[:index], p.disconnected[index+1:]...)
	if delete {
		p.dead = append(p.dead, backend)
	}
	return backend, nil
}

// dead -> delete
func (p *ServerPool) Delete(rawUrl string) (*Backend, error) {
	index, ok := serverIndex(p.dead, rawUrl)
	if !ok {
		return nil, ErrNoSuchUrl
	}

	backend := p.dead[index]
	p.dead = append(p.dead[:index], p.dead[index+1:]...)
	return backend, nil
}

// disconnected -> alive
func (p *ServerPool) Resurrect(rawUrl string) error {
	backend, err := p.Disconnect(rawUrl, false)
	if err != nil {
		return err
	}
	backend.Retries = 0
	p.Add(backend)
	return nil
}

type CopyResult struct {
	resultCh chan []*Backend
}

type BackendResult struct {
	resultCh chan<- *Backend
}

func NewGateway() *Gateway {
	gtw := Gateway{
		serverPool:         NewServerPool(),
		policy:             &RoundRobinPolicy{},
		syncCh:             make(chan struct{}),
		addCh:              make(chan *Backend),
		removeCh:           make(chan string),
		eraseCh:            make(chan string),
		disconnectCh:       make(chan string),
		deleteCh:           make(chan string),
		resurrectCh:        make(chan string),
		nextCh:             make(chan *BackendResult),
		copyAliveCh:        make(chan *CopyResult),
		copyDisconnectedCh: make(chan *CopyResult),
		copyDeadCh:         make(chan *CopyResult),
	}

	go gtw.process()

	return &gtw
}

type Gateway struct {
	serverPool         *ServerPool
	policy             Policy
	monitorUrl         string
	syncCh             chan struct{}
	addCh              chan *Backend
	removeCh           chan string
	eraseCh            chan string
	deleteCh           chan string
	disconnectCh       chan string
	resurrectCh        chan string
	copyAliveCh        chan *CopyResult
	copyDeadCh         chan *CopyResult
	copyDisconnectedCh chan *CopyResult
	nextCh             chan *BackendResult
}

func (g *Gateway) process() {
	var backend *Backend
	var rawUrl string
	var backendResult *BackendResult
	var copyResult *CopyResult

	for {
		select {
		case <-g.syncCh:
			// no-op

		case backend = <-g.addCh:
			g.serverPool.Add(backend)

		case rawUrl = <-g.removeCh:
			g.serverPool.Remove(rawUrl)

		case rawUrl = <-g.eraseCh:
			g.serverPool.Erase(rawUrl)

		case rawUrl = <-g.disconnectCh:
			g.serverPool.Disconnect(rawUrl, true)

		case rawUrl = <-g.deleteCh:
			g.serverPool.Delete(rawUrl)

		case rawUrl = <-g.resurrectCh:
			g.serverPool.Resurrect(rawUrl)

		case copyResult = <-g.copyAliveCh:
			copied := make([]*Backend, len(g.serverPool.alives))
			copy(copied, g.serverPool.alives)
			copyResult.resultCh <- copied

		case copyResult = <-g.copyDisconnectedCh:
			copied := make([]*Backend, len(g.serverPool.disconnected))
			copy(copied, g.serverPool.disconnected)
			copyResult.resultCh <- copied

		case copyResult = <-g.copyDeadCh:
			copied := make([]*Backend, len(g.serverPool.dead))
			copy(copied, g.serverPool.dead)
			copyResult.resultCh <- copied

		case backendResult = <-g.nextCh:
			if len(g.serverPool.alives) == 0 {
				backendResult.resultCh <- nil
			} else {
				backendResult.resultCh <- g.next()
			}
		}
	}
}

func (g *Gateway) sync() {
	g.syncCh <- struct{}{}
}

func (g *Gateway) Add(backend *Backend) {
	g.addCh <- backend
	g.sync()
}

func (g *Gateway) Remove(rawUrl string) {
	g.removeCh <- rawUrl
	g.sync()
}

func (g *Gateway) Erase(rawUrl string) {
	g.eraseCh <- rawUrl
	g.sync()
}

func (g *Gateway) Disconnect(rawUrl string) {
	g.disconnectCh <- rawUrl
	g.sync()
}

func (g *Gateway) Delete(rawUrl string) {
	g.deleteCh <- rawUrl
	g.sync()
}

func (g *Gateway) Resurrect(rawUrl string) {
	g.resurrectCh <- rawUrl
	g.sync()
}

func (g *Gateway) next() *Backend {
	return g.policy.Select(g.serverPool)
}

func (g *Gateway) copyAlive() []*Backend {
	copyResult := CopyResult{
		resultCh: make(chan []*Backend),
	}
	g.copyAliveCh <- &copyResult
	return <-copyResult.resultCh
}

func (g *Gateway) copyDisconnected() []*Backend {
	copyResult := CopyResult{
		resultCh: make(chan []*Backend),
	}
	g.copyDisconnectedCh <- &copyResult
	return <-copyResult.resultCh
}

func (g *Gateway) copyDead() []*Backend {
	copyResult := CopyResult{
		resultCh: make(chan []*Backend),
	}
	g.copyDeadCh <- &copyResult
	return <-copyResult.resultCh
}

func (g *Gateway) healthCheck() {
	backends := g.copyAlive()
	var wg sync.WaitGroup
	wg.Add(len(backends))
	for _, b := range backends {
		go func(b *Backend) {
			rawUrl := b.rawUrl()
			alive := isBackendAlive(rawUrl + "/health_check")
			if !alive {
				g.Remove(rawUrl)
				b.Retries++
			}
			wg.Done()
		}(b)
	}
	wg.Wait()
}

func (g *Gateway) resurrectDisconnected() {
	backends := g.copyDisconnected()
	for _, b := range backends {
		rawUrl := b.rawUrl()
		alive := isBackendAlive(rawUrl + "/health_check")
		if alive {
			g.Resurrect(rawUrl)
		} else if b.Retries >= 3 {
			g.Disconnect(rawUrl)
		} else {
			b.Retries++
		}
	}
}

func (g *Gateway) report() error {
	dead := g.copyDead()

	log.Printf("Removed Servers:%d\n", len(dead))

	deadServers := make([]string, len(dead))
	for _, v := range dead {
		deadServers = append(deadServers, v.rawUrl())
	}

	err := reportDeadBackends(g.monitorUrl, deadServers)
	if err != nil {
		log.Println(err)
		return err
	}

	for _, v := range dead {
		g.Delete(v.rawUrl())
	}

	return nil
}

func (g *Gateway) NextServer() *Backend {
	resultCh := make(chan *Backend)
	backendResult := &BackendResult{
		resultCh: resultCh,
	}

	g.nextCh <- backendResult
	select {
	case backend := <-resultCh:
		return backend
	case <-time.After(100 * time.Millisecond):
	}

	return nil
}
