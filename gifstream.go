package main

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
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
	ImageIndex map[string]*Img
	ImageKeys  []string
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

type Broadcaster struct {
	// Using a map like a set
	Listeners map[*Listener]struct{}
}

type Listener struct {
	Stream chan string
}

var (
	imageStore  = NewImageStore()
	broadcaster = NewBroadcaster()
	templates   = template.Must(template.ParseFiles(
		"home.html",
	))
)

func (store *ImageStore) Start() {
	// Loop forever, processing each type of request
	for {
		select {
		case request := <-store.allChannel:
			var all []string = make([]string, len(store.ImageKeys))
			copy(all, store.ImageKeys)
			request.ResponseChan <- all
		case request := <-store.getChannel:
			request.ResponseChan <- store.ImageIndex[request.Key]
		case request := <-store.putChannel:
			store.ImageIndex[request.Key] = request.Image
			store.ImageKeys = append(store.ImageKeys, request.Key)
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
	store.ImageIndex = make(map[string]*Img)
	store.ImageKeys = make([]string, 0)

	// A channel for each request type
	store.getChannel = make(chan *GetImageRequest)
	store.putChannel = make(chan *PutImageRequest)
	store.allChannel = make(chan *AllImageRequest)

	return store
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		Listeners: make(map[*Listener]struct{}),
	}
}

func (broadcaster *Broadcaster) NewListener() *Listener {
	listener := &Listener{
		Stream: make(chan string),
	}

	broadcaster.Listeners[listener] = struct{}{}

	return listener
}

func (broadcaster *Broadcaster) Remove(listener *Listener) {
	delete(broadcaster.Listeners, listener)
}

func (broadcaster *Broadcaster) Send(key string) {
	for listener := range broadcaster.Listeners {
		listener.Stream <- key
	}
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
		log.Println(err.Error())

		return nil, err
	}

	var pngImage bytes.Buffer
	err = png.Encode(&pngImage, g)
	if err != nil {
		log.Println(err.Error())

		return nil, err
	}
	image.Snapshot = pngImage

	return image, nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "home.html", imageStore.All())
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusFound)

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

		return
	}

	key := keyFor(buf.Bytes())

	// Store the image in memory
	imageStore.Put(key, img)

	// Broadcast new image to everyone
	broadcaster.Send(key)

	http.Redirect(w, r, "/", http.StatusFound)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-type", "image/png")
		w.Write(img.Snapshot.Bytes())
	} else {
		// Serve the animated gif
		w.Header().Set("Content-type", "image/gif")
		w.Write(img.Original.Bytes())
	}

}

func streamHandler(ws *websocket.Conn) {
	listener := broadcaster.NewListener()
	for {
		imgKey := <-listener.Stream
		log.Printf("Sending message: %s", imgKey)
		err := websocket.Message.Send(ws, imgKey)
		if err != nil {
			broadcaster.Remove(listener)
			log.Println(err.Error())

			break
		}
	}
}

func main() {
	// imageStore is a global
	go imageStore.Start()

	log.Println("Listening for requests...")

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/img", imageHandler)
	http.Handle("/stream", websocket.Handler(streamHandler))
	http.ListenAndServe(":5555", nil)
}
