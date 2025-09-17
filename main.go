package main

import (
	"bytes"
	ddos "cloudflare-clone/DDoS"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	body       []byte
	header     http.Header
	statusCode int
	expiry     time.Time
}

func (e cacheEntry) isExpired() bool {
	return time.Now().After(e.expiry)
}

type Cache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
}

func NewCache() *Cache {
	return &Cache{data: make(map[string]cacheEntry)}
}

func (c *Cache) Get(path string) (cacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.data[path]
	if !ok || entry.isExpired() {
		return cacheEntry{}, false
	}
	return entry, true
}

func (c *Cache) Set(path string, entry cacheEntry) {
	c.mu.Lock()
	c.data[path] = entry
	c.mu.Unlock()
}

func (c *Cache) EvictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for path, entry := range c.data {
		if entry.isExpired() {
			delete(c.data, path)
			log.Println("🗑️ Evicted expired cache:", path)
		}
	}
}

type CacheProxy struct {
	origin *url.URL
	proxy  *httputil.ReverseProxy
	cache  *Cache
}

func NewCacheProxy(originURL string) (*CacheProxy, error) {
	origin, err := url.Parse(originURL)
	if err != nil {
		return nil, err
	}

	cp := &CacheProxy{
		origin: origin,
		proxy:  httputil.NewSingleHostReverseProxy(origin),
		cache:  NewCache(),
	}

	cp.proxy.ModifyResponse = cp.cacheResponse
	return cp, nil
}

func (cp *CacheProxy) cacheResponse(r *http.Response) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body.Close()

	entry := cacheEntry{
		body:       body,
		header:     r.Header.Clone(),
		statusCode: r.StatusCode,
		expiry:     expiryForContentType(r.Header.Get("Content-Type")),
	}

	cp.cache.Set(r.Request.URL.Path, entry)

	r.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

func (cp *CacheProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		log.Println("Not caching method:", r.Method)
		cp.proxy.ServeHTTP(w, r)
		return
	}

	if entry, found := cp.cache.Get(r.URL.Path); found {
		for k, v := range entry.header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
		w.WriteHeader(entry.statusCode)
		w.Write(entry.body)
		log.Println("⚡ Cache HIT for", r.URL.Path)
		return
	}

	log.Println("➡️ Cache MISS for", r.URL.Path)
	cp.proxy.ServeHTTP(w, r)
}

func (cp *CacheProxy) StartEvictionLoop(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			cp.cache.EvictExpired()
		}
	}()
}

func expiryForContentType(contentType string) time.Time {
	now := time.Now()
	switch {
	case strings.HasPrefix(contentType, "text/html"):
		return now.Add(5 * time.Second)
	case strings.HasPrefix(contentType, "image/"):
		return now.Add(1 * time.Minute)
	case strings.HasPrefix(contentType, "application/json"):
		return now.Add(15 * time.Second)
	case strings.HasPrefix(contentType, "video/"):
		return now.Add(1 * time.Minute)
	default:
		return now.Add(10 * time.Second)
	}
}

func main() {
	counter := ddos.NewDDoS(5)

	limiter := ddos.NewDDoSLimiter(3, 3*time.Second)

	cp, err := NewCacheProxy("http://localhost:8080")
	if err != nil {
		log.Fatal("Error creating proxy:", err)
	}
	cp.StartEvictionLoop(1 * time.Second)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ipStr, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Invalid IP", http.StatusBadRequest)
			return
		}
		ip := net.ParseIP(ipStr)

		counter.AddRequest(ip)
		if counter.IsSuspicious(ip) {
			fmt.Println("Suspicious traffic from:", ip)
		}

		limiter.AllowRequest(ipStr)

		cp.ServeHTTP(w, r)
	})

	fmt.Println("🚀 Server running on :8443")
	if err := http.ListenAndServeTLS(":8443", "cert.pem", "key.pem", handler); err != nil {
		log.Fatal("Server error:", err)
	}
}
