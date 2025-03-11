package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := "8082"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Healthy")
	})

	// Main endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
		
		fmt.Fprintf(w, "Response from backend server on port %s\n", port)
		fmt.Fprintf(w, "Request path: %s\n", r.URL.Path)
		fmt.Fprintf(w, "Request method: %s\n", r.Method)
		fmt.Fprintf(w, "Request headers: %v\n", r.Header)
	})

	log.Printf("Backend server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
