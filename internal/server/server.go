package server

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/readflow/readflow/internal/handler"
	readflowmcp "github.com/readflow/readflow/internal/mcp"
	"github.com/readflow/readflow/internal/middleware"
	"github.com/readflow/readflow/internal/store"
	"github.com/readflow/readflow/internal/store/sessionstore"
)

type Server struct {
	store *store.Store
	http  *http.Server
}

func New(dbPath string) (*Server, error) {
	s, err := store.New(dbPath)
	if err != nil {
		return nil, err
	}

	migrationFiles := []string{"001_init.sql", "002_webauthn.sql"}
	for _, f := range migrationFiles {
		migrationsSQL, err := os.ReadFile(filepath.Join(findProjectRoot(), "migrations", f))
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", f, err)
		}
		if err := s.Migrate(string(migrationsSQL)); err != nil {
			return nil, fmt.Errorf("run migration %s: %w", f, err)
		}
	}

	sm := scs.New()
	sm.Store = sessionstore.New(s.DB())
	sm.Lifetime = 7 * 24 * time.Hour
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode

	if os.Getenv("READFLOW_ENV") == "production" {
		sm.Cookie.Secure = true
	}

	tmpl := loadTemplates()
	h := handler.New(s, tmpl, sm)

	mcpServer := readflowmcp.NewServer(s, h.SaveArticle)
	mcpHTTP := mcpserver.NewStreamableHTTPServer(mcpServer)

	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.RedactAPIKeyQuery)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecureHeaders)

	fileServer := http.FileServer(http.Dir(filepath.Join(findProjectRoot(), "static")))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	r.Route("/", func(r chi.Router) {
		r.Use(sm.LoadAndSave)

		r.Get("/setup", h.SetupPage)
		r.With(middleware.PerIPRateLimit(0.5, 5)).Post("/setup", h.Setup)

		r.Get("/login", h.LoginPage)
		r.With(middleware.PerIPRateLimit(0.5, 5)).Post("/login", h.Login)

		// Passkey (WebAuthn) login endpoints, available before authentication.
		r.Post("/webauthn/login/begin", h.WebAuthnLoginBegin)
		r.Post("/webauthn/login/finish", h.WebAuthnLoginFinish)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthRequired(sm))

			r.Get("/", h.Index)
			r.Get("/read/{id}", h.ReadPage)
			r.Get("/read/{id}/iframe", h.ReadIframePage)
			r.Get("/save", h.SavePage)
			r.Get("/settings", h.SettingsPage)
			r.Get("/logout", h.Logout)

			r.Post("/save", h.SaveForm)
			r.Post("/save-link", h.SaveLink)
			r.Post("/generate-key", h.GenerateAPIKey)
			r.Post("/delete-key/{id}", h.DeleteAPIKey)
			r.Post("/generate-webhook-token", h.GenerateWebhookToken)
			r.Post("/delete-webhook-token/{id}", h.DeleteWebhookToken)
			r.Post("/delete/{id}", h.DeleteArticle)
			r.Post("/delete-batch", h.DeleteArticles)

			// Passkey (WebAuthn) registration and management.
			r.Post("/webauthn/register/begin", h.WebAuthnRegisterBegin)
			r.Post("/webauthn/register/finish", h.WebAuthnRegisterFinish)
			r.Get("/webauthn/credentials", h.WebAuthnCredentials)
			r.Delete("/webauthn/credentials/{id}", h.WebAuthnCredentialDelete)
		})
	})

	r.With(
		middleware.PerIPRateLimit(10, 20),
		middleware.WebhookTokenAuth(s),
	).Post("/hooks/inoreader", h.InoreaderWebhook)

	// MCP endpoint for AI assistants, authenticated with an API key.
	r.With(
		middleware.PerIPRateLimit(10, 20),
		middleware.APIKeyAuth(s),
	).Handle("/mcp", mcpHTTP)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.PerIPRateLimit(10, 20))
		r.Use(middleware.APIKeyAuth(s))

		r.Post("/save", h.APISave)
		r.Post("/webhooks/inoreader", h.InoreaderWebhook)
		r.Get("/export", h.APIExport)
		r.Get("/article/{id}", h.APIReadArticle)
		r.Get("/read/{id}", h.ReadMobilePage)
		r.Post("/delete", h.APIDeleteArticles)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{store: s, http: httpServer}, nil
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func loadTemplates() *template.Template {
	templateDir := filepath.Join(findProjectRoot(), "template")

	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"formatTime": func(s string) string {
			if len(s) >= 10 {
				return s[:10]
			}
			return s
		},
		"truncate": func(s string, n int) string {
			if len(s) > n {
				return s[:n] + "..."
			}
			return s
		},
		"add":      func(a, b int) int { return a + b },
		"subtract": func(a, b int) int { return a - b },
		"multiply": func(a, b int) int { return a * b },
	}

	tmpl := template.New("").Funcs(funcMap)
	pattern := filepath.Join(templateDir, "*.html")
	files, _ := filepath.Glob(pattern)
	if len(files) == 0 {
		panic("no template files found in " + templateDir)
	}
	return template.Must(tmpl.ParseFiles(files...))
}

func findProjectRoot() string {
	cwd, _ := os.Getwd()
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd
		}
		dir = parent
	}
}
