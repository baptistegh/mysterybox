package server

import (
	"compress/gzip"
	"context"
	"encoding/json"
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

func NextMonday(t time.Time) time.Time {
	// Calculate days until next Monday
	daysUntilMonday := (int(time.Monday) - int(t.Weekday()) + 7) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7 // If today is Monday, get next Monday
	}

	nextMonday := t.AddDate(0, 0, daysUntilMonday)

	// Return next Monday at 00:00:00 in the same location
	return time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 0, 0, 0, 0, t.Location())
}

func WeeksPassed(now time.Time, from time.Time) int8 {
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
			weekNumber int8   = 0
			nextUpdate string = NextMonday(now).Format(time.ANSIC)
			rTitle            = "Encore un peu de patience"
			rText             = "Le jeux commencera prochainement.."
		)

		if now.After(conf.StartDate) {
			index := WeeksPassed(now, conf.StartDate)
			weekNumber = index + 1
			if index >= int8(len(conf.Riddles)) {
				index = int8(len(conf.Riddles) - 1)
			}
			currentRiddle := conf.Riddles[index]
			rTitle = currentRiddle.Title
			rText = currentRiddle.Text
		}

		layouts.Base("Mystery Box", "homepage", weekNumber, nextUpdate, rTitle, rText).Render(r.Context(), w)
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
