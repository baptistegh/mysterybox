package server

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
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
	Title string `json:"title"`
	Text  string `json:"text"`
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
		now := time.Now()

		var (
			nextUpdate time.Time = NextUpdate(now, conf.StartDate)
			weekNumber int8      = WeeksPassed(now, conf.StartDate)
			rTitle               = "Encore un peu de patience"
			rText                = "Le jeux commencera prochainement.."
		)

		log.Default().Printf("now=%s, startDate=%s, nextUpdate=%s, weekNumber=%d", now, conf.StartDate.String(), nextUpdate.String(), weekNumber)

		if now.After(conf.StartDate) {
			if weekNumber >= int8(len(conf.Riddles)) {
				weekNumber = int8(len(conf.Riddles) - 1)
			}
			currentRiddle := conf.Riddles[weekNumber]
			rTitle = currentRiddle.Title
			rText = currentRiddle.Text
		}

		if weekNumber <= 0 {
			weekNumber = -1
		}

		layouts.Base("Mystery Box", "homepage", weekNumber+1, nextUpdate.Format(time.ANSIC), rTitle, rText).Render(r.Context(), w)
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
