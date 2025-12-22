package server

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/baptistegh/mysterybox/server/middleware"
	"github.com/baptistegh/mysterybox/views"
	"github.com/baptistegh/mysterybox/views/layouts"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	server *http.Server
}

type Handler interface {
	Register(*http.ServeMux)
}

type Conf struct {
	StartDate time.Time `json:"start_date"`
	Riddles   []Riddle  `json:"riddles"`
}

type Riddle struct {
	Title  string `json:"title"`
	Text   string `json:"text"`
	Answer string `json:"answer"`
}

func Riddles() (*Conf, error) {
	fp, err := os.Open("./riddles.json")
	if err != nil {
		return nil, err
	}

	var riddles Conf
	if err = json.NewDecoder(fp).Decode(&riddles); err != nil {
		return nil, err
	}

	return &riddles, nil
}

func NextUpdate(now time.Time, start time.Time) time.Time {
	next := start
	for {
		if next.After(now) {
			return next
		}
		next = next.AddDate(0, 0, 7)
	}
}

func WeeksPassed(now time.Time, from time.Time) int8 {
	if now.Before(from) {
		log.Default().Println("The game has not started yet!")
		return -1
	}
	duration := now.Sub(from)
	return int8(duration.Hours() / 24 / 7)
}

func NewServer(addr string) (*Server, error) {
	conf, err := Riddles()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("GET /assets/", middleware.AssetsCache(http.FileServer(views.Assets)))
	mux.Handle("GET /", Root())
	mux.HandleFunc("GET /home", func(w http.ResponseWriter, r *http.Request) {
		var (
			rTitle = "Jouons à un petit jeu!"
			rText  = "Trouve toutes les énigmes pour ouvrire cette boîte mystérieuse."
		)

		// If the request comes from HTMX, return only the fragment so we don't swap a full page
		if r.Header.Get("HX-Request") == "true" {
			layouts.Home(rTitle, rText).Render(r.Context(), w)
			return
		}

		layouts.Base("Mystery Box", "homepage", rTitle, rText).Render(r.Context(), w)
	})

	mux.HandleFunc("GET /riddles/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		i, err := strconv.ParseInt(id, 10, 8)
		if err != nil {
			log.Default().Printf("Error: wrong id given %s, %s", id, err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if i < 0 || int(i) >= len(conf.Riddles) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var (
			rTitle = conf.Riddles[i].Title
			rText  = conf.Riddles[i].Text
		)

		layouts.Riddle(id, rTitle, rText, "").Render(r.Context(), w)
	})

	mux.HandleFunc("POST /riddles/{id}/answer", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		i, err := strconv.ParseInt(id, 10, 8)
		if err != nil {
			log.Default().Printf("Error: wrong id given %s, %s", id, err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if i < 0 || int(i) >= len(conf.Riddles) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		answer := strings.TrimSpace(strings.ToLower(r.FormValue("answer")))
		expected := strings.TrimSpace(strings.ToLower(conf.Riddles[i].Answer))

		if expected == "" {
			layouts.Riddle(id, conf.Riddles[i].Title, conf.Riddles[i].Text, "Aucune réponse configurée pour cette énigme.").Render(r.Context(), w)
			return
		}

		if answer == expected {
			nextIndex := int(i) + 1
			if nextIndex >= len(conf.Riddles) {
				layouts.End().Render(r.Context(), w)
				return
			}
			layouts.Riddle(strconv.FormatInt(int64(nextIndex), 10), conf.Riddles[nextIndex].Title, conf.Riddles[nextIndex].Text, "").Render(r.Context(), w)
			return
		}

		// wrong answer
		layouts.Riddle(id, conf.Riddles[i].Title, conf.Riddles[i].Text, "Mauvaise réponse, essaie encore.").Render(r.Context(), w)
	})

	compressor := chimiddleware.NewCompressor(gzip.DefaultCompression)

	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: chimiddleware.Recoverer(middleware.Logger(compressor.Handler(mux))),
		},
	}, nil
}

func (s *Server) Run() error {
	return s.server.ListenAndServe()
}

func Root() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			return
		}
		http.Redirect(w, r, "/home", http.StatusTemporaryRedirect)
	})
}

func (s *Server) Stop(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.server.Shutdown(ctx)
}
