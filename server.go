package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"github.com/drone/routes"
	"io"
	"net/http"
	"os"
	"strconv"
)

var totalBytes int64

var stream cipher.Stream

func main() {
	initCipher()
	initTotalBytes()

	mux := routes.New()
	mux.Get("/", http.HandlerFunc(handleLandingPage))
	mux.Get("/totalbytes", http.HandlerFunc(handleTotalBytes))
	mux.Get("/blob", http.HandlerFunc(handleBlob))
	mux.Get("/blob/:size([0-9]*)", http.HandlerFunc(handleBlob))
	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)
}

// Init

func initTotalBytes() {
	file, err := os.Open("totalbytes") // For read access.
	if err != nil {
		panic(err)
	}
	fmt.Fscanln(file, "%d", &totalBytes)
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
	fmt.Fprintf(w, "%d", totalBytes)
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

	totalBytes += int64(size)
}

// --

func writeTotalBytes(chan int64 c) {

}
