package entrysvc

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type Server struct {
	instance string
	db       *sql.DB
	router   *httprouter.Router
	entries  *EntryService
}

func NewServer(instance string, entryService *EntryService) *Server {
	s := &Server{
		instance: instance,
		router:   httprouter.New(),
		entries:  entryService,
	}
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) Respond(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("error: %v", err)
	}
}

func (s *Server) Decode(w http.ResponseWriter, r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func (s *Server) Error(w http.ResponseWriter, r *http.Request, status int, msg string) {
	s.Respond(w, r, status, map[string]string{"error": msg})
}

func (s *Server) registerRoutes() {
	s.router.GET("/live", s.handleLive())
	s.router.GET("/ready", s.handleReady())
	s.router.POST("/entries", s.handleEntryPost())
	s.router.GET("/entries", s.handleEntrySearch())
}

// always respond with ok as long as the server is running
func (s *Server) handleLive() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"instance": "%s", "alive": true}`, s.instance)
	}
}

// only respond with ok if the database is reachable
func (s *Server) handleReady() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if err := s.entries.Ready(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			log.Printf("db ping failed: %v", err)
			fmt.Fprintf(w, `{"instance": "%s", "ready": false}`, s.instance)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"instance": "%s", "ready": true}`, s.instance)
	}
}

func (s *Server) handleEntryPost() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var entry Entry
		if err := s.Decode(w, r, &entry); err != nil {
			log.Printf("error decoding entry: %v", err)
			s.Error(w, r, http.StatusBadRequest, "invalid entry")
			return
		}
		if err := s.entries.Add(r.Context(), &entry); err != nil {
			if err == ErrNoValue {
				s.Error(w, r, http.StatusUnprocessableEntity, "missing value")
				return
			}
			log.Printf("error posting entry: %v", err)
			s.Error(w, r, http.StatusInternalServerError, "error posting entry")
			return
		}
		s.Respond(w, r, http.StatusCreated, entry)
	}
}

func (s *Server) handleEntrySearch() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		entries, err := s.entries.List(r.Context())
		if err != nil {
			log.Printf("error searching entries: %v", err)
			s.Error(w, r, http.StatusInternalServerError, "error searching entries")
			return
		}
		s.Respond(w, r, http.StatusOK, entries)
	}
}
