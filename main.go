package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	jwtCookieName                 = "JWT"
	contextUsernameKey contextKey = "context_username"
)

func (s *server) authMiddleware(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(jwtCookieName)
		if err != nil {
			slog.Error("read JWT cookie", "error", err)
			Login("", "").Render(r.Context(), w)
			return
		}
		if cookie == nil {
			slog.Error("missing JWT cookie")
			Login("", "").Render(r.Context(), w)
			return
		}

		token, err := s.verifyToken(cookie.Value)
		if err != nil {
			slog.Error("verifying token", "error", err)
			Login("", "").Render(r.Context(), w)
			return
		}

		subj, err := token.Claims.GetSubject()
		if err != nil {
			slog.Error("getting token subject", "error", err)
			Login("", "").Render(r.Context(), w)
			return
		}

		ctx := context.WithValue(r.Context(), contextUsernameKey, subj)
		r = r.WithContext(ctx)

		slog.Info("got token", "val", token.Raw)

		next(w, r)
	}
}

func (s *server) createToken(username string) (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"iss": "templ-demo",
		"exp": time.Now().Add(time.Second * 20).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := claims.SignedString([]byte(s.jwtSecretKey))
	if err != nil {
		return "", fmt.Errorf("sign claims: %w", err)
	}

	return tokenString, nil
}

func (s *server) verifyToken(tokenString string) (*jwt.Token, error) {
	// Parse the token with the secret key
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecretKey), nil
	})

	// Check for verification errors
	if err != nil {
		return nil, err
	}

	// Check if the token is valid
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Return the verified token
	return token, nil
}

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
	mux.HandleFunc("/", srv.authMiddleware(srv.handleHome))
	mux.HandleFunc("/login", srv.handleLogin)
	mux.HandleFunc("/signup", srv.handleSignup)
	mux.HandleFunc("/account", srv.authMiddleware(srv.handleAccount))

	http.ListenAndServe(":3000", mux)
}

type server struct {
	db           *database
	jwtSecretKey string
}

func newServer(db *database) (*server, error) {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "TEST_KEY"
	}
	return &server{
		db:           db,
		jwtSecretKey: secretKey,
	}, nil
}

func (s *server) handleHome(w http.ResponseWriter, r *http.Request) {
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

	token, err := s.createToken(username)
	if err != nil {
		slog.Error("failed to create JWT", "error", err)
		Home().Render(r.Context(), w)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  jwtCookieName,
		Value: token,
	})

	ctx := context.WithValue(r.Context(), contextUsernameKey, username)
	r = r.WithContext(ctx)

	Home().Render(r.Context(), w)
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

	token, err := s.createToken(username)
	if err != nil {
		slog.Error("failed to create JWT", "error", err)
		Home().Render(r.Context(), w)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  jwtCookieName,
		Value: token,
	})

	ctx := context.WithValue(r.Context(), contextUsernameKey, username)
	r = r.WithContext(ctx)

	Home().Render(r.Context(), w)
}

func (s *server) handleAccount(w http.ResponseWriter, r *http.Request) {
	Account().Render(r.Context(), w)
}

//go:embed public/styles.css
var cssFile []byte

func serveCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write(cssFile)
}
