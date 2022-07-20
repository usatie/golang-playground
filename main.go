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

var c Cache

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

func panicHandler(w http.ResponseWriter, r *http.Request) {
	panic("expected panic!")
}

func initCache() {
	c.m = sync.Map{}
	c.ttl = 10 * time.Second
}

func main() {
	t := time.Now().UnixNano()
	initCache()
	rand.Seed(t)
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", handler)
	serveMux.HandleFunc("/fortune", getFortune)
	serveMux.HandleFunc("/slow", slowHandler)
	serveMux.HandleFunc("/cached", cachedHandler)
	serveMux.Handle("/foo", handle{})
	serveMux.HandleFunc("/panic", panicHandler)
	srv := &http.Server{
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      2 * time.Second,
		Addr:              ":8080",
		Handler:           http.TimeoutHandler(serveMux, 2*time.Second, ""),
	}
	log.Println(srv.ListenAndServe())
}
