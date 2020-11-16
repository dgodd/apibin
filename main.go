package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dgodd/apibin/pubsub"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	eb := pubsub.NewEventBus()
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/watch/{id:[a-f09]+}", func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadFile("public/watch.html")
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
		w.Write(b)
	})

	r.Get("/sse/{id:[a-f09]+}", func(w http.ResponseWriter, r *http.Request) {
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
			fmt.Fprintf(w, "data: %s\n\n", d.Data)
			flusher.Flush()
		}
	})

	r.Post("/post/{id:[a-f09]+}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		fmt.Println(id, body)

		eb.Publish(id, body)
	})

	http.ListenAndServe(":3000", r)
}
