package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

var group singleflight.Group

func getFortune(w http.ResponseWriter, r *http.Request) {
	n := rand.Intn(6)
	var res string
	switch n + 1 {
	case 1:
		res = "凶"
	case 2, 3:
		res = "小吉"
	case 4, 5:
		res = "中吉"
	case 6:
		res = "大吉"
	}
	fmt.Fprintf(w, "%s", res)
}
func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, HTTP server")
}

func slowHandler(w http.ResponseWriter, r *http.Request) {
	ch := make(chan int)
	quit := make(chan int)
	go func() {
		for i := 0; i < 5; i++ {
			select {
			case <-quit:
				return
			default:
				time.Sleep(1 * time.Second)
			}

		}
		fmt.Fprint(w, "This message should not appear")
		log.Println("5 seconds passed.")
		ch <- 0
	}()
	select {
	case <-r.Context().Done():
		log.Println("it's canceled.")
		quit <- 0
		return
	case <-ch:
		log.Println("it's completed.")
		return
	}
}

type handle struct{}

func (h handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler(w, r)
	fmt.Fprint(w, " from handle")
}

type item struct {
	data     interface{}
	ttl      time.Duration
	expireAt time.Time
}

func newItem(data interface{}, ttl time.Duration) *item {
	item := &item{
		data:     data,
		ttl:      ttl,
		expireAt: time.Now().Add(ttl),
	}
	item.touch()
	return item
}

func (i *item) touch() {
	i.expireAt = time.Now().Add(i.ttl)
}

func (item *item) expired() bool {
	return item.expireAt.Before(time.Now())
}

type Cache struct {
	mu sync.RWMutex
	m  map[string]*item
}

var c Cache

func (c *Cache) Get(key string) interface{} {
	c.mu.RLock()
	v, ok := c.m[key]
	c.mu.RUnlock()
	if ok && !v.expired() {
		return v.data
	}
	c.mu.Lock()
	go func() {
		data := heavyGet(key)
		// この2行を
		c.m[key] = newItem(data, ttl)
		c.mu.Unlock()
		// これに変えると動かない。なぜならSetの最初のLock()がとれないから
		// c.Set(key, data)
	}()
	return c.Get(key)
}

var ttl time.Duration = 10 * time.Second

func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	c.m[key] = newItem(value, ttl)
	c.mu.Unlock()
}
func heavyGet(key string) int {
	time.Sleep(1 * time.Second)
	return len(key)
}
func cachedHandler(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	v := c.Get(key)
	fmt.Fprintln(w, v.(int))
}

func main() {
	t := time.Now().UnixNano()
	m := make(map[string]*item)
	c = Cache{m: m}
	rand.Seed(t)
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", handler)
	serveMux.HandleFunc("/fortune", getFortune)
	serveMux.HandleFunc("/slow", slowHandler)
	serveMux.HandleFunc("/cached", cachedHandler)
	serveMux.Handle("/foo", handle{})
	srv := &http.Server{
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      2 * time.Second,
		Addr:              ":8080",
		Handler:           http.TimeoutHandler(serveMux, 2*time.Second, ""),
	}
	log.Println(srv.ListenAndServe())
}
