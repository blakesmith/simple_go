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
	request := new(PutImageRequest)
	request.Key = key
	request.ResponseChan = make(chan string)
	request.Image = img
	store.putChannel <- request

	return <-request.ResponseChan
}

func (store *ImageStore) Get(key string) *Img {
	request := new(GetImageRequest)
	request.Key = key
	request.ResponseChan = make(chan *Img)
	store.getChannel <- request

	return <-request.ResponseChan
}

func (store *ImageStore) All() []string {
	request := new(AllImageRequest)
	request.ResponseChan = make(chan []string)
	store.allChannel <- request

	return <-request.ResponseChan
}

func NewImageStore() *ImageStore {
	store := new(ImageStore)
	store.Images = make(map[string]*Img)
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
	image.Buffer = buf

	return image, nil
}

func uploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		images := imageStore.All()
		templates.ExecuteTemplate(w, "upload.html", images)

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

	imageStore.Put(keyFor(buf.Bytes()), img)

	http.Redirect(w, r, "/", http.StatusFound)
}

func displayImage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.Form.Get("key")
	img := imageStore.Get(key)

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
	go imageStore.Start()

	log.Println("Listening for requests...")

	http.HandleFunc("/img", displayImage)
	http.HandleFunc("/", uploadImage)
	http.ListenAndServe(":5555", nil)
}
