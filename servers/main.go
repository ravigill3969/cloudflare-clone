package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hello from the ORIGIN server 🚀"))
	})

	mux.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "example.txt")
	})

	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]string{"origin": "true", "message": "Hello from JSON!"}
		json.NewEncoder(w).Encode(data)
	})

	mux.HandleFunc("/video", func(w http.ResponseWriter, r *http.Request) {
		videoFile := "./files/video.mp4"
		f, err := os.Open(videoFile)
		if err != nil {
			http.Error(w, "Video not found", http.StatusNotFound)
			return
		}
		defer f.Close()
		info, _ := f.Stat()
		w.Header().Set("Content-Type", "video/mp4")
		http.ServeContent(w, r, videoFile, info.ModTime(), f)
	})

	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		http.ServeFile(w, r, "./files/image.webp")
	})

	log.Println("📡 Origin server running on http://localhost:8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
