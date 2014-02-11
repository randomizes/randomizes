package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

var totalBytes int64

var channel chan int64

var pool []byte

var serviceAvailable int32 = 0 // Porcaria do go n tem atomic.toogleBool ...

func main() {

	initTotalBytes()

	fmt.Print("Initializing entropy generator - ")
	initEntropyGenerator()
	fmt.Println("DONE")

	mux := mux.NewRouter()
	mux.HandleFunc("/", http.HandlerFunc(handleLandingPage))
	mux.HandleFunc("/totalbytes", http.HandlerFunc(handleTotalBytes))
	mux.HandleFunc("/blob", http.HandlerFunc(handleBlob))
	mux.HandleFunc("/blob/{size:[0-9]+}", http.HandlerFunc(handleBlob))

	http.Handle("/", mux)
	http.ListenAndServe(":3000", nil)

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

func initEntropyGenerator() {

	block, err := aes.NewCipher([]byte("jiboias ao poder"))
	if err != nil {
		panic(err)
	}

	var iv [aes.BlockSize]byte
	stream := cipher.NewOFB(block, iv[:])

	pool = make([]byte, 1024, 1024)

	// Plain Generator
	t1 := time.NewTicker(time.Second * 3)
	go func() {
		for _ = range t1.C {

			fmt.Println("gen() - FETCHING")

			res, err := http.Get("https://github.com/timeline.json")
			if err != nil {
				fmt.Println("ERROR: ", err)
				atomic.StoreInt32(&serviceAvailable, 0)
				continue
			}

			reader := &cipher.StreamReader{S: stream, R: res.Body}
			io.ReadFull(reader, pool)

			atomic.StoreInt32(&serviceAvailable, 1)

			fmt.Println("gen() - DONE")
		}
	}()

	// Shufffler
	t2 := time.NewTicker(time.Second * 1)
	go func() {
		fmt.Println("suffler() - STARTED")
		for _ = range t2.C {
			pool[rand.Intn(1024)] = pool[rand.Intn(1024)]
			fmt.Println("suffler() - SHUFFLE")
		}
	}()

}

// Handlers

func handleLandingPage(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, "./index.html")
}

func handleTotalBytes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
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
	params := mux.Vars(r)
	size, err := strconv.Atoi(params["size"])

	if size < 0 || err != nil {
		size = 64
	}

	if size > 1024 {
		size = 1024
	}

	fmt.Printf("handleBlob() - REQUEST %d BYTES\n", size)

	start := rand.Intn(1024)
	if start+size > 1024 {
		w.Write(pool[start:])
		w.Write(pool[:size-(1024-start)])
		fmt.Printf("handleBlob() - START: [%d : 2014] - END [0 : %d]\n", start, size-(1024-start))
	} else {
		w.Write(pool[start:size])
		fmt.Printf("handleBlob() - START: [%d : %d]\n", start, size)
	}

	channel <- int64(size)
}

// --

func writeTotalBytes(file *os.File) {
	for {
		totalBytes += <-channel

		err := file.Truncate(0)
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
