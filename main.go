package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	bolt "go.etcd.io/bbolt"
)

var (
	addr   string
	dbPath string
	db     *bolt.DB
)

func init() {
	flag.StringVar(&addr, "addr", ":8080", "listen addr for the HTTP server")
	flag.StringVar(&dbPath, "db-path", "hourlymotion.db", "path of the local database storage")

	http.Handle("/", http.FileServer(http.Dir("")))
	http.HandleFunc("/event", storeEvent)
	http.HandleFunc("/displayAd", displayAd)
	http.HandleFunc("/admin-data", adminData)
	http.HandleFunc("/user-data", userData)
	http.HandleFunc("/user-settings", userSettings)
}

func main() {
	flag.Parse()

	log.Printf("Opening database at %s\n", dbPath)
	var err error
	db, err = bolt.Open(dbPath, 0600, nil)
	checkErr(err)
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(k("hourlymotion"))
		return err
	})
	checkErr(err)

	log.Printf("Starting server on %s\n", addr)
	log.Println(http.ListenAndServe(addr, nil))
}

func storeEvent(w http.ResponseWriter, r *http.Request) {
	var (
		event Event
		err   error
	)
	if err = json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// not logged-in user
	if len(event.User.DailymotionID) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		var userData UserData
		bucket := tx.Bucket(k("hourlymotion"))
		data := bucket.Get(k(event.User.DailymotionID))
		if len(data) == 0 {
			userData = UserData{
				Xid: event.User.DailymotionID,
			}
		} else {
			if err := json.Unmarshal(data, &userData); err != nil {
				return err
			}
		}
		switch event.Type {
		case "ad_start":
			userData.Ads++
			userData.Tokens++
		case "video_start":
			userData.Videos++
		}
		log.Printf("Storing/Updating event %s with user data %#v", event.Type, userData)
		data, err := json.Marshal(userData)
		if err != nil {
			return err
		}
		return bucket.Put(k(event.User.DailymotionID), data)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func displayAd(w http.ResponseWriter, r *http.Request) {
	userXid := r.URL.Query().Get("userXid")
	if len(userXid) == 0 {
		http.Error(w, "missing userXid query parameter", http.StatusBadRequest)
		return
	}
	videoXid := r.URL.Query().Get("videoXid")
	if len(videoXid) == 0 {
		http.Error(w, "missing videoXid query parameter", http.StatusBadRequest)
		return
	}

	displayAd, err := shouldDisplayAd(userXid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%t", displayAd)
}

func shouldDisplayAd(userXid string) (bool, error) {
	var displayAd bool
	err := db.Update(func(tx *bolt.Tx) error {
		var userData UserData
		bucket := tx.Bucket(k("hourlymotion"))
		data := bucket.Get(k(userXid))
		if len(data) == 0 {
			userData = UserData{
				Xid: userXid,
			}
		} else {
			if err := json.Unmarshal(data, &userData); err != nil {
				return err
			}
		}

		if userData.Tokens > 0 {
			displayAd = false
		} else {
			displayAd = true
		}

		if !displayAd {
			log.Printf("will NOT display an ad for user %#v", userData)
			userData.Tokens--
			userData.UsedTokens++
		}

		log.Printf("(maybe) updated user data %#v", userData)
		data, err := json.Marshal(userData)
		if err != nil {
			return err
		}
		return bucket.Put(k(userXid), data)
	})
	return displayAd, err
}

func adminData(w http.ResponseWriter, r *http.Request) {
	usersData := []UserData{}
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(k("hourlymotion"))
		return bucket.ForEach(func(k, v []byte) error {
			var userData UserData
			if err := json.Unmarshal(v, &userData); err != nil {
				return err
			}
			usersData = append(usersData, userData)
			return nil
		})
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err = json.NewEncoder(w).Encode(usersData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func userData(w http.ResponseWriter, r *http.Request) {
	userXid := r.URL.Query().Get("userXid")
	if len(userXid) == 0 {
		http.Error(w, "missing userXid query parameter", http.StatusBadRequest)
		return
	}

	var userData *UserData
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(k("hourlymotion"))
		data := bucket.Get(k(userXid))
		if len(data) > 0 {
			userData = new(UserData)
			if err := json.Unmarshal(data, userData); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if userData == nil {
		http.NotFound(w, r)
		return
	}
	if err = json.NewEncoder(w).Encode(userData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func userSettings(w http.ResponseWriter, r *http.Request) {
	userXid := r.URL.Query().Get("userXid")
	if len(userXid) == 0 {
		http.Error(w, "missing userXid query parameter", http.StatusBadRequest)
		return
	}

	var (
		settings Settings
		err      error
	)
	if err = json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		var userData UserData
		bucket := tx.Bucket(k("hourlymotion"))
		data := bucket.Get(k(userXid))
		if len(data) == 0 {
			userData = UserData{
				Xid: userXid,
			}
		} else {
			if err := json.Unmarshal(data, &userData); err != nil {
				return err
			}
		}
		userData.Settings = settings
		log.Printf("Storing/Updating settings %#v for user %s", settings, userXid)
		data, err := json.Marshal(userData)
		if err != nil {
			return err
		}
		return bucket.Put(k(userXid), data)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

type UserData struct {
	Xid        string
	Ads        int64
	Videos     int64
	Tokens     int64
	UsedTokens int64
	Settings   Settings
}

type Settings struct {
	AutoUseTokens string
}

type User struct {
	DailymotionID string `json:"xid"`
}

type Event struct {
	User    User   `json:"user"`
	Type    string `json:"type"`
	VideoID string `json:"video_id"`
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func getenvOrDefault(name, defaultValue string) string {
	if value := os.Getenv(name); len(value) > 0 {
		return value
	}
	return defaultValue
}

func k(str string) []byte {
	return []byte(str)
}
