package main

import (
	_ "embed"
	"errors"
	"log/slog"
	"net/http"
)

func main() {
	db, err := newDatabase("./test.db")
	if err != nil {
		slog.Error("creating database", "error", err)
		return
	}
	srv, err := newServer(db)
	if err != nil {
		slog.Error("create new server", "error", err)
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/public/styles.css", serveCSS)
	mux.HandleFunc("/", srv.handleHome)
	mux.HandleFunc("/login", srv.handleLogin)
	mux.HandleFunc("/signup", srv.handleSignup)

	http.ListenAndServe(":3000", mux)
}

type server struct {
	db *database
}

func newServer(db *database) (*server, error) {

	return &server{db: db}, nil
}

func (s *server) handleHome(w http.ResponseWriter, r *http.Request) {
	// TODO: check for JWT
	if r.FormValue("override") == "true" {
		Home("test user").Render(r.Context(), w)
		return
	}
	Login("", "").Render(r.Context(), w)
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	err := s.db.loginUser(username, password)
	if err != nil {
		if errors.Is(err, errUserNotFound) {
			Signup(username, password, "").Render(r.Context(), w)
			return
		}

		if errors.Is(err, errIncorrectPassword) {
			Login(username, err.Error()).Render(r.Context(), w)
			return
		}

		slog.Error("failed to log user in", "error", err)
		http.Error(w, "failed to log in", http.StatusInternalServerError)
		return
	}

	Home(username).Render(r.Context(), w)
}

func (s *server) handleSignup(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm-password")
	if password != confirmPassword {
		slog.Info("passwords don't match")
		Signup(username, password, "passwords don't match").Render(r.Context(), w)
		return
	}

	err := s.db.createUser(username, password)
	if err != nil {
		slog.Error("failed to create user", "error", err)
		Signup(username, password, err.Error()).Render(r.Context(), w)
		return
	}

	Home(username).Render(r.Context(), w)
}

//go:embed public/styles.css
var cssFile []byte

func serveCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write(cssFile)
}
