package main

import (
	"fmt"
	"log"
	"net/http"
)

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, Golang!")
}

func main() {
	log.Println("Listening for requests...")

	http.HandleFunc("/", hello)
	http.ListenAndServe(":5555", nil)
}