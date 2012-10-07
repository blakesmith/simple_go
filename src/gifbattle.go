package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"image/gif"
	"image/png"
	"io"
	"log"
	"net/http"
	"text/template"
)

type Img struct {
	Original bytes.Buffer
	Snapshot bytes.Buffer
}

type ImageStore struct {
	Images     map[string]*Img
	getChannel chan *GetImageRequest
	putChannel chan *PutImageRequest
	allChannel chan *AllImageRequest
}

type GetImageRequest struct {
	Key          string
	ResponseChan chan *Img
}

type PutImageRequest struct {
	Key          string
	Image        *Img
	ResponseChan chan string
}

type AllImageRequest struct {
	ResponseChan chan []string
}

var (
	imageStore = NewImageStore()
	templates  = template.Must(template.ParseFiles(
		"upload.html",
	))
)

func (store *ImageStore) Start() {
	// Loop forever, processing each type of request
	for {
		select {
		case request := <-store.allChannel:
			keys := make([]string, 0)

			for key := range store.Images {
				keys = append(keys, key)
			}

			request.ResponseChan <- keys
		case request := <-store.getChannel:
			request.ResponseChan <- store.Images[request.Key]
		case request := <-store.putChannel:
			store.Images[request.Key] = request.Image
			request.ResponseChan <- fmt.Sprintf("%s: %s", "OK", request.Key)
		}
	}
}

func (store *ImageStore) Put(key string, img *Img) string {
	request := &PutImageRequest{
		Key:          key,
		ResponseChan: make(chan string),
		Image:        img,
	}
	store.putChannel <- request

	return <-request.ResponseChan
}

func (store *ImageStore) Get(key string) *Img {
	request := &GetImageRequest{
		Key:          key,
		ResponseChan: make(chan *Img),
	}
	store.getChannel <- request

	return <-request.ResponseChan
}

func (store *ImageStore) All() []string {
	request := &AllImageRequest{
		ResponseChan: make(chan []string),
	}
	store.allChannel <- request

	return <-request.ResponseChan
}

func NewImageStore() *ImageStore {
	store := new(ImageStore)
	store.Images = make(map[string]*Img)

	// A channel for each request type
	store.getChannel = make(chan *GetImageRequest)
	store.putChannel = make(chan *PutImageRequest)
	store.allChannel = make(chan *AllImageRequest)

	return store
}

func keyFor(b []byte) string {
	sha := sha1.New()
	sha.Write(b)

	return fmt.Sprintf("%x", string(sha.Sum(nil))[0:10])
}

func decodeGif(buf bytes.Buffer) (*Img, error) {
	image := new(Img)
	image.Original = buf

	var newBuf bytes.Buffer
	io.Copy(&newBuf, &buf)

	g, err := gif.Decode(&newBuf)
	if err != nil {
		log.Fatal(err)

		return nil, err
	}

	var pngImage bytes.Buffer
	err = png.Encode(&pngImage, g)
	if err != nil {
		log.Fatal(err)

		return nil, err
	}
	image.Snapshot = pngImage

	return image, nil
}

func uploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		images := imageStore.All()
		templates.ExecuteTemplate(w, "upload.html", images)

		return
	}

	file, _, err := r.FormFile("image")

	if err != nil {
		log.Println(err.Error())
		http.Redirect(w, r, "/", http.StatusFound)

		return
	}
	// Close the file at the end of this function
	defer file.Close()

	var buf bytes.Buffer
	io.Copy(&buf, file)

	// Snapshot and extract image
	img, err := decodeGif(buf)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}

	// Store the image in memory
	imageStore.Put(keyFor(buf.Bytes()), img)

	http.Redirect(w, r, "/", http.StatusFound)
}

func displayImage(w http.ResponseWriter, r *http.Request) {
	// Read the url params
	r.ParseForm()
	key := r.Form.Get("key")
	img := imageStore.Get(key)

	// No image specified
	if img == nil {
		http.Error(w, "Image not found!", 400)

		return
	}
	w.Header().Set("Cache-Control", "max-age=3600, public")

	if r.Form.Get("thumb") == "true" {
		// Serve the still thumbnail
		w.Header().Set("Content-type", "image/gif")
		w.Write(img.Snapshot.Bytes())
	} else {
		// Serve the animated gif
		w.Header().Set("Content-type", "image/png")
		w.Write(img.Original.Bytes())
	}

}

func main() {
	go imageStore.Start()

	log.Println("Listening for requests...")

	http.HandleFunc("/img", displayImage)
	http.HandleFunc("/", uploadImage)
	http.ListenAndServe(":5555", nil)
}
