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

func main() {
	t := time.Now().UnixNano()
	initCache()
	rand.Seed(t)
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/omikuji", getOmikuji)
	serveMux.HandleFunc("/cached", cachedHandler)
	serveMux.HandleFunc("/panic", panicHandler)
	serveMux.Handle("/handle", handle{})
	serveMux.HandleFunc("/handler", handler)
	serveMux.HandleFunc("/slow", slowHandler)
	srv := &http.Server{
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      2 * time.Second,
		Addr:              ":8080",
		Handler:           http.TimeoutHandler(serveMux, 2*time.Second, ""),
	}
	log.Println(srv.ListenAndServe())
}

// GET /omikuji
func getOmikuji(w http.ResponseWriter, r *http.Request) {
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

// GET /cached
var (
	group singleflight.Group
	c     Cache
)

func initCache() {
	c.m = sync.Map{}
	c.ttl = 10 * time.Second
}

func heavyGet(key string) int {
	time.Sleep(1 * time.Second)
	return len(key)
}
func cachedHandler(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	if v, err := c.Get(key); err == nil {
		fmt.Fprintln(w, v.(int))
		return
	}
	groupKey := fmt.Sprintf("cachedHandler.%s", key)
	v, err, _ := group.Do(groupKey, func() (interface{}, error) {
		val := heavyGet(key)
		c.Set(key, val)
		return val, nil
	})
	if err != nil {
		return
	}
	fmt.Fprintln(w, v.(int))
}

// GET /panic
func panicHandler(w http.ResponseWriter, r *http.Request) {
	panic("expected panic!")
}

// GET /handler
func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "response from handler")
}

// GET /handle
type handle struct{}

func (h handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler(w, r)
	fmt.Fprint(w, "response from handle")
}

// GET /slow
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
