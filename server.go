package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	_ "github.com/drone/routes"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

var totalBytes int64

var stream cipher.Stream

var channel chan int64

func main() {

	initCipher()
	initTotalBytes()

	mux := mux.NewRouter()
	mux.HandleFunc("/", http.HandlerFunc(handleLandingPage))
	mux.HandleFunc("/totalbytes", http.HandlerFunc(handleTotalBytes))
	mux.HandleFunc("/blob", http.HandlerFunc(handleBlob))
	mux.HandleFunc("/blob/:size([0-9]*)", http.HandlerFunc(handleBlob))

	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)

}

// Init

func initTotalBytes() {
	file, err := os.OpenFile("totalbytes", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	fmt.Fscanf(file, "%d", &totalBytes)

	fmt.Println("totalBytes: ", totalBytes)

	channel = make(chan int64, 128)

	go writeTotalBytes(file)
}

func initCipher() {
	key := []byte("jiboias ao poder")

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	var iv [aes.BlockSize]byte
	stream = cipher.NewOFB(block, iv[:])
}

// Handlers

func handleLandingPage(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, "./index.html")
}

func handleTotalBytes(w http.ResponseWriter, r *http.Request) {
	for {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%d", totalBytes)
		flusher.Flush()

		time.Sleep(1 * time.Second)
	}
}

func handleBlob(w http.ResponseWriter, r *http.Request) {
	res, err := http.Get("https://github.com/timeline.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	size, err := strconv.Atoi(r.URL.Query().Get(":size"))
	if err != nil {
		size = 64
	}

	if size > 1024 {
		size = 1024
	}

	reader := &cipher.StreamReader{S: stream, R: res.Body}
	if _, err := io.CopyN(w, reader, int64(size)); err != nil {
		panic(err)
	}

	channel <- int64(size)
}

// --

func writeTotalBytes(file *os.File) {
	for {
		totalBytes += <-channel

		_, err := file.Seek(0, 0)
		if err != nil {
			fmt.Println(err)
		}

		//_, err :=
		fmt.Fprintf(file, "%d", totalBytes)
		// if err != nil {
		// 	fmt.Println(err)
		// }

		file.Sync()
		fmt.Println("file written: ", totalBytes)
	}
}
