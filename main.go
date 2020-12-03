package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/dgodd/apibin/pubsub"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type HeadBody struct {
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
}

func main() {
	eb := pubsub.NewEventBus()
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/", func(r chi.Router) {
		basicAuthPasswords := map[string]string{os.Getenv("AUTH_USERNAME"): os.Getenv("AUTH_PASSWORD")}
		r.Use(middleware.BasicAuth("API Bin", basicAuthPasswords))

		r.Get("/watch/{id:[a-z0-9]+}", func(w http.ResponseWriter, r *http.Request) {
			b, err := ioutil.ReadFile("public/watch.html")
			if err != nil {
				http.Error(w, "Error reading file", http.StatusInternalServerError)
				return
			}
			w.Write(b)
		})

		r.Get("/sse/{id:[a-z0-9]+}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming not supported", http.StatusNotAcceptable)
				return
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			ch := make(chan pubsub.DataEvent)
			eb.Subscribe(id, ch)

			for {
				d := <-ch
				d2, _ := json.Marshal(d.Data)
				fmt.Fprintf(w, "data: %s\n\n", d2)
				flusher.Flush()
			}
		})
	})

	r.Post("/post/{id:[a-z0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		fmt.Println(id, r.Header, body)

		eb.Publish(id, HeadBody{Header: r.Header, Body: string(body)})
	})

	http.ListenAndServe(":3333", r)
}
