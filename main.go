package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

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

type handle struct{}

func (h handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler(w, r)
	fmt.Fprint(w, " from handle")
}

func main() {
	t := time.Now().UnixNano()
	rand.Seed(t)
	http.HandleFunc("/", handler)
	http.HandleFunc("/fortune", getFortune)
	http.Handle("/foo", handle{})
	http.ListenAndServe(":8080", nil)
}
