package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"sync"
  //  "math/rand"
	"fmt"
	"database/sql"
	_ "github.com/lib/pq"
)

var (
	addr       string
	events     []Event
	eventsLock sync.Mutex
	postgre		*PostgreClient
)

const UPDATE_VIDEO_WATCHED = "update dm_user set videos_watched = videos_watched + 1, tokens_to_redeem = tokens_to_redeem+1 where user_id = $1"
const UPDATE_AD_WATCHED = "update dm_user set ads_watched = ads_watched + 1, tokens_to_redeem = tokens_to_redeem+1 where user_id = $1"
const SELECT_REDEEM_COUNT = "select reduce_ads, tokens_to_redeem from dm_user where user_id = $1"
const UPDATE_REDEEM_COUNT = "update dm_user set tokens_to_redeem = tokens_to_redeem - 4 where user_id = $1"


func init() {
	flag.StringVar(&addr, "addr", ":8080", "listen addr for the HTTP server")

	postgre = NewPostgreClient()

	http.Handle("/", http.FileServer(http.Dir("")))
	http.HandleFunc("/event", storeEvent)
	http.HandleFunc("/events", listEvents)
    http.HandleFunc("/displayAd", displayAd)
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

	if (event.Type == "video_end") || (event.Type == "ad_end") && (event.User.DailymotionID != "")  {
		db := postgre.getDbConnection()
		defer db.Close()

		query := UPDATE_VIDEO_WATCHED

		if event.Type == "ad_end" {
			query = UPDATE_AD_WATCHED
		}

		db.QueryRow(query, event.User.DailymotionID)

	}

	w.WriteHeader(http.StatusCreated)

	/* eventsLock.Lock()
	events = append(events, event)
	eventsLock.Unlock() */


}

func listEvents(w http.ResponseWriter, r *http.Request) {
	eventsLock.Lock()
	defer eventsLock.Unlock()

	if err := json.NewEncoder(w).Encode(&events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func displayAd(w http.ResponseWriter, r *http.Request) {
    userXids, ok := r.URL.Query()["userXid"]
    
    if !ok || len(userXids[0]) < 1 {
        log.Println("Url Param 'userXid' is missing")
        return
    }

    userXid := userXids[0]

    log.Println("UserXid ", userXid);
    
    videoXids, ok := r.URL.Query()["videoXid"]
    
    if !ok || len(videoXids[0]) < 1 {
        log.Println("Url Param 'videoXids' is missing")
        return
    }
    
    videoXid := videoXids[0]

	log.Println("videoXid ", videoXid);

	db := postgre.getDbConnection()
	defer db.Close()

	rows, err := db.Query(SELECT_REDEEM_COUNT, userXid)
	checkErr(err)

	for rows.Next() {
		var reduce_ads bool
		var tokens_to_redeem int
		err = rows.Scan(&reduce_ads, &tokens_to_redeem)
		checkErr(err)
		if (reduce_ads == true) && (tokens_to_redeem > 4) {
			db.QueryRow(UPDATE_REDEEM_COUNT, userXid)
			fmt.Fprintf(w, "true")
		} else {
			fmt.Fprintf(w, "false")
		}
	}

	w.WriteHeader(http.StatusOK);

    /* randomNum := rand.Intn(10)
    
    if randomNum >= 8 {
        fmt.Fprintf(w, "false")
    } else {
        fmt.Fprintf(w, "true")
    } */

}

type User struct {
	DailymotionID string `json:"xid"`
}

type Event struct {
	User    User   `json:"user"`
	Type    string `json:"type"`
	VideoID string `json:"video_id"`
}

type PostgreClient struct {

}

func NewPostgreClient() *PostgreClient {
	return &PostgreClient{}
}


func (pc *PostgreClient) getDbConnection() *sql.DB  {
	dbinfo := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		"hourlymotion.cgwqhqmxi2gt.us-east-1.rds.amazonaws.com", "hourlyadmin", "DMhackathon2018", "hourlymotiondb")
	db, err := sql.Open("postgres", dbinfo)
	checkErr(err)
	return db
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
