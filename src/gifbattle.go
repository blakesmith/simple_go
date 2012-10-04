package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func hello(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("hello_golang.png")
	if err != nil {
		log.Fatal(err)

		return
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)

		return
	}

	w.Header().Set("Content-type", "image/png")
	w.Write(b)
}

func uploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Must be a POST", 400)
		return
	}

	file, _, err := r.FormFile("image")
	checkError(err)
	defer file.Close()

	var buf bytes.Buffer
	io.Copy(&buf, file)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	log.Println("Listening for requests...")

	http.HandleFunc("/", hello)
	http.HandleFunc("/upload", uploadImage)
	http.ListenAndServe(":5555", nil)
}
