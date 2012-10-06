package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"text/template"
)

type Img struct {
	Buffer   bytes.Buffer
	Original image.Image
}

var (
	AllImages map[string]*Img = make(map[string]*Img)
	templates                 = template.Must(template.ParseFiles(
		"upload.html",
	))
)

func keyFor(b []byte) string {
	sha := sha1.New()
	sha.Write(b)

	return fmt.Sprintf("%x", string(sha.Sum(nil))[0:10])
}

func decodeGif(buf bytes.Buffer) (*Img, error) {
	image := new(Img)
	image.Buffer = buf

	return image, nil
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

	log.Println(AllImages)
	fmt.Fprintf(w, "OK")
}

func displayImage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	key := r.Form.Get("key")

	img := AllImages[key]

	if img == nil {
		http.Error(w, "Image not found!", 400)

		return
	}

	w.Header().Set("Content-type", "image/gif")
	w.Write(img.Buffer.Bytes())
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	log.Println("Listening for requests...")

	http.HandleFunc("/img", displayImage)
	http.HandleFunc("/", uploadImage)
	http.ListenAndServe(":5555", nil)
}
