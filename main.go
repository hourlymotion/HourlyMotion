package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"sync"
)

var (
	addr       string
	events     []Event
	eventsLock sync.Mutex
)

func init() {
	flag.StringVar(&addr, "addr", ":8080", "listen addr for the HTTP server")

	http.Handle("/", http.FileServer(http.Dir("")))
	http.HandleFunc("/event", storeEvent)
	http.HandleFunc("/events", listEvents)
}

func main() {
	flag.Parse()

	log.Printf("Starting server on %s\n", addr)
	log.Println(http.ListenAndServe(addr, nil))
}

func storeEvent(w http.ResponseWriter, r *http.Request) {
	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	eventsLock.Lock()
	events = append(events, event)
	eventsLock.Unlock()

	w.WriteHeader(http.StatusCreated)
}

func listEvents(w http.ResponseWriter, r *http.Request) {
	eventsLock.Lock()
	defer eventsLock.Unlock()

	if err := json.NewEncoder(w).Encode(&events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type User struct {
	DailymotionID string `json:"xid"`
}

type Event struct {
	User    User   `json:"user"`
	Type    string `json:"type"`
	VideoID string `json:"video_id"`
}
