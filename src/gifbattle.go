package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"image"
	"image/gif"
	"io"
	"log"
	"net/http"
	"text/template"
)

var (
	AllImages map[string]image.Image = make(map[string]image.Image)
	templates                        = template.Must(template.ParseFiles(
		"upload.html",
	))
)

func keyFor(b []byte) string {
	sha := sha1.New()
	sha.Write(b)

	return fmt.Sprintf("%x", string(sha.Sum(nil))[0:10])
}

func decodeGif(buf bytes.Buffer) (image.Image, error) {
	img, err := gif.Decode(&buf)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func uploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		templates.ExecuteTemplate(w, "upload.html", nil)

		return
	}

	file, _, err := r.FormFile("image")

	checkError(err)
	defer file.Close()

	var buf bytes.Buffer
	io.Copy(&buf, file)

	img, err := decodeGif(buf)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}

	AllImages[keyFor(buf.Bytes())] = img

	fmt.Fprintf(w, "OK")
}

func editForm(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "edit.html", r.FormValue("id"))
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	log.Println("Listening for requests...")

	http.HandleFunc("/upload", uploadImage)
	http.ListenAndServe(":5555", nil)
}
