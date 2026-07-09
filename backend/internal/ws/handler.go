package ws

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	"caipiao/backend/internal/auth"
	"caipiao/backend/internal/middleware"
)

type Server struct {
	Hub     *Hub
	Auth    *auth.Service
	Origins []string
}

func (s *Server) upgrader() *websocket.Upgrader {
	origins := s.Origins
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return middleware.IsAllowedOrigin(r.Header.Get("Origin"), r.Host, origins)
		},
	}
}

func (s *Server) HandlePublic(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.Hub == nil {
		http.Error(w, "websocket unavailable", http.StatusServiceUnavailable)
		return
	}
	conn, err := s.upgrader().Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := newConn(s.Hub, conn, KindPublic)
	go c.Run(nil)
}

func (s *Server) HandleClient(w http.ResponseWriter, r *http.Request) {
	s.handleAuthed(w, r, KindClient)
}

func (s *Server) HandleAdmin(w http.ResponseWriter, r *http.Request) {
	s.handleAuthed(w, r, KindAdmin)
}

func (s *Server) handleAuthed(w http.ResponseWriter, r *http.Request, kind ConnKind) {
	if s == nil || s.Hub == nil || s.Auth == nil {
		http.Error(w, "websocket unavailable", http.StatusServiceUnavailable)
		return
	}
	conn, err := s.upgrader().Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := newConn(s.Hub, conn, kind)
	authFn := func(token string) (string, bool) {
		if token == "" {
			token = strings.TrimSpace(r.URL.Query().Get("token"))
		}
		if token == "" {
			return "", false
		}
		claims, err := s.Auth.ParseBearer(token)
		if err != nil {
			return "", false
		}
		want := auth.RoleClient
		if kind == KindAdmin {
			want = auth.RoleAdmin
		}
		if claims.Role != want {
			return "", false
		}
		return claims.Subject, true
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		if account, ok := authFn(token); ok {
			c.authenticated = true
			c.account = account
		}
	}
	go c.Run(authFn)
}