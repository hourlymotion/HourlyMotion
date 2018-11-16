# HourlyMotion

* `go run main.go` -> start an HTTP server on port `8080` by default
* `ngrok http 8080` -> expose your local server to the outside world

Application URLs:
* `/index.html`: watch a video
* `/admin-dashboard.html`: display all stored data
* `/user-dashboard.html`: display stored data for the logged-in user

API endpoints:
* `POST /event`: store a new event - from a JSON payload
* `GET /displayAd`: accepts `userXid` and `videoXid`, returns `true`/`false` if it should display an ad or not
* `GET /admin-data`: returns a JSON array of all data stored for all users
* `GET /user-data`: accepts `userXid` and return a JSON object of the data stored for the user
* `POST /user-settings`: accepts `userXid` and a JSON payload of the user's settings to store

if you want to also start our custom ad-director:
* get https://github.com/vbehar/ad-director/tree/hourlymotion
  * `git clone https://github.com/vbehar/ad-director.git $GOPATH/src/github.com/dailymotion/ad-director`
  * `git checkout hourlymotion`
* run `make ad-director` to build it
* edit the `ad-director.conf` conf file, mainly the `hourlymotion` section - with your own hourlymotion endpoint
* run `sudo ./ad-director ad-director.conf` to start it. need `sudo` because we'll need to start it on port `443`
* edit your `/etc/hosts` file, and add `127.0.0.1  dmxleo.dailymotion.com` so that requests for ad-director hit your own instance
* make sure to trust the self-signed certificate located at `dmxleo.dailymotion.com.pem`
