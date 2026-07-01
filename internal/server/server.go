package server

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/readflow/readflow/internal/handler"
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

	migrationsSQL, err := os.ReadFile(filepath.Join(findProjectRoot(), "migrations", "001_init.sql"))
	if err != nil {
		return nil, err
	}
	if err := s.Migrate(string(migrationsSQL)); err != nil {
		return nil, err
	}

	sm := scs.New()
	sm.Store = sessionstore.New(s.DB())
	sm.Lifetime = 7 * 24 * time.Hour
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode

	if os.Getenv("READFLOW_ENV") == "production" {
		sm.Cookie.Secure = true
	}

	if os.Getenv("READFLOW_ENV") == "production" {
		sm.Cookie.Secure = true
	}

	tmpl := loadTemplates()
	h := handler.New(s, tmpl, sm)

	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecureHeaders)

	fileServer := http.FileServer(http.Dir(filepath.Join(findProjectRoot(), "static")))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	r.Route("/", func(r chi.Router) {
		r.Use(sm.LoadAndSave)

		r.Get("/setup", h.SetupPage)
		r.Post("/setup", h.Setup)

		r.Get("/login", h.LoginPage)
		r.Post("/login", h.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthRequired(sm))

			r.Get("/", h.Index)
			r.Get("/read/{id}", h.ReadPage)
			r.Get("/read/{id}/iframe", h.ReadIframePage)
			r.Get("/save", h.SavePage)
			r.Get("/settings", h.SettingsPage)
			r.Get("/logout", h.Logout)

			r.Post("/save", h.SaveForm)
			r.Post("/generate-key", h.GenerateAPIKey)
			r.Post("/delete-key/{id}", h.DeleteAPIKey)
			r.Post("/archive/{id}", h.ArchiveArticle)
			r.Post("/delete/{id}", h.DeleteArticle)
			r.Post("/delete-batch", h.DeleteArticles)
			r.Post("/unread/{id}", h.UnreadArticle)
		})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.RateLimit(10, 20))
		r.Use(middleware.APIKeyAuth(s))

		r.Post("/save", h.APISave)
		r.Get("/export", h.APIExport)
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
		"add": func(a, b int) int { return a + b },
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
