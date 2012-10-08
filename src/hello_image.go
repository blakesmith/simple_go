package main

import (
	"log"
	"net/http"
	"io/ioutil"
	"os"
)

func hello(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("hello_golang.png")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-type", "image/png")
	w.Write(b)
}

func main() {
	log.Println("Listening for requests...")

	http.HandleFunc("/", hello)
	http.ListenAndServe(":5555", nil)
}
