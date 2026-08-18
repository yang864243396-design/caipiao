package ws

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	"caipiao/backend/internal/auth"
	"caipiao/backend/internal/middleware"
)

type Server struct {
	Hub                   *Hub
	Auth                  *auth.Service
	Origins               []string
	ResolveClientIdentity func(context.Context, string) (ClientIdentity, error)
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
	identityContext := context.WithoutCancel(r.Context())
	authFn := func(token string) (ClientIdentity, bool) {
		if token == "" {
			token = strings.TrimSpace(r.URL.Query().Get("token"))
		}
		if token == "" {
			return ClientIdentity{}, false
		}
		claims, err := s.Auth.ParseBearer(token)
		if err != nil {
			return ClientIdentity{}, false
		}
		want := auth.RoleClient
		if kind == KindAdmin {
			want = auth.RoleAdmin
		}
		if claims.Role != want {
			return ClientIdentity{}, false
		}
		if kind == KindClient {
			if s.ResolveClientIdentity == nil {
				return ClientIdentity{}, false
			}
			identity, err := s.ResolveClientIdentity(identityContext, claims.Subject)
			if err != nil || strings.TrimSpace(identity.Account) != strings.TrimSpace(claims.Subject) || identity.MemberID <= 0 {
				return ClientIdentity{}, false
			}
			return identity, true
		}
		return ClientIdentity{Account: claims.Subject}, true
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		if identity, ok := authFn(token); ok {
			_ = s.Hub.BindClientIdentity(c, identity)
		}
	}
	go c.Run(authFn)
}
