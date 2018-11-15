# HourlyMotion

* `go run main.go` -> start an HTTP server on port `8080` by default
* `ngrok http 8080` -> expose your local server to the outside world

API endpoints:
* `POST /event`: store a new event
* `GET /events`: returns a JSON list of all stored events
