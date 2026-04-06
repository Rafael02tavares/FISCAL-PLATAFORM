package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/auth"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/catalog"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/companies"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/config"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/fiscaloperations"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/invoices"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/legalbasis"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/ncm"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/organizations"
	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/tax"
)

type Server struct {
	cfg  config.Config
	db   *pgxpool.Pool
	mux  *http.ServeMux
	cors map[string]struct{}
}

func New(cfg config.Config, db *pgxpool.Pool) http.Handler {
	s := &Server{
		cfg: cfg,
		db:  db,
		mux: http.NewServeMux(),
		cors: map[string]struct{}{
			"http://localhost:4321": {},
		},
	}

	s.registerRoutes()

	return s.withCORS(s.mux)
}

func (s *Server) registerRoutes() {
	s.registerHealthRoutes()
	s.registerPublicRoutes()
	s.registerProtectedRoutes()
}

func (s *Server) registerHealthRoutes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
}

func (s *Server) registerPublicRoutes() {
	// auth
	authRepo := auth.NewRepository(s.db)
	authService := auth.NewService(authRepo)
	jwtService := auth.NewJWT(s.cfg.JWTSecret)
	authHandler := auth.NewHandler(authService, jwtService)

	s.mux.HandleFunc("POST /auth/register", authHandler.Register)
	s.mux.HandleFunc("POST /auth/login", authHandler.Login)

	// companies
	companyClient := companies.NewClient()
	companyService := companies.NewService(companyClient)
	companyHandler := companies.NewHandler(companyService)
	s.mux.HandleFunc("GET /companies/lookup", companyHandler.Lookup)

	// fiscal operations
	fiscalOpRepo := fiscaloperations.NewRepository(s.db)
	fiscalOpService := fiscaloperations.NewService(fiscalOpRepo)
	fiscalOpHandler := fiscaloperations.NewHandler(fiscalOpService)
	s.mux.HandleFunc("GET /fiscal-operations", fiscalOpHandler.List)

	// ncm
	ncmRepo := ncm.NewRepository(s.db)
	ncmService := ncm.NewService(ncmRepo)
	ncmHandler := ncm.NewHandler(ncmService)

	s.mux.HandleFunc("GET /ncm", ncmHandler.List)
	s.mux.HandleFunc("GET /ncm/find", ncmHandler.GetByCode)
	s.mux.HandleFunc("GET /ncm/search", ncmHandler.Search)

	// legal basis
	legalRepo := legalbasis.NewRepository(s.db)
	legalService := legalbasis.NewService(legalRepo)
	legalHandler := legalbasis.NewHandler(legalService)

	s.mux.HandleFunc("GET /legal-sources", legalHandler.ListLegalSources)
	s.mux.HandleFunc("POST /legal-sources", legalHandler.CreateLegalSource)
	s.mux.HandleFunc("GET /legal-rules", legalHandler.ListLegalRuleMappings)
	s.mux.HandleFunc("POST /legal-rules", legalHandler.CreateLegalRuleMapping)
}

func (s *Server) registerProtectedRoutes() {
	protectedMux := http.NewServeMux()

	// dependencies
	authRepo := auth.NewRepository(s.db)
	authService := auth.NewService(authRepo)
	jwtService := auth.NewJWT(s.cfg.JWTSecret)
	authHandler := auth.NewHandler(authService, jwtService)

	orgRepo := organizations.NewRepository(s.db)
	orgService := organizations.NewService(orgRepo)
	orgHandler := organizations.NewHandler(orgService)

	catalogRepo := catalog.NewRepository(s.db)
	catalogService := catalog.NewService(catalogRepo)

	invoiceRepo := invoices.NewRepository(s.db)
	invoiceService := invoices.NewService(invoiceRepo, catalogService)
	invoiceHandler := invoices.NewHandler(invoiceService, orgService)

	fiscalOpRepo := fiscaloperations.NewRepository(s.db)
	fiscalOpService := fiscaloperations.NewService(fiscalOpRepo)

	legalRepo := legalbasis.NewRepository(s.db)
	legalService := legalbasis.NewService(legalRepo)

	taxRepo := tax.NewRepository(s.db)
	taxService := tax.NewService(taxRepo, fiscalOpService, legalService)
	taxHandler := tax.NewHandler(taxService, orgService)

	// routes
	protectedMux.HandleFunc("GET /auth/me", authHandler.Me)

	protectedMux.HandleFunc("POST /organizations", orgHandler.Create)
	protectedMux.HandleFunc("GET /organizations", orgHandler.List)

	protectedMux.HandleFunc("POST /invoices/upload", invoiceHandler.Upload)
	protectedMux.HandleFunc("GET /invoices", invoiceHandler.List)
	protectedMux.HandleFunc("GET /invoices/", invoiceHandler.GetByID)

	protectedMux.HandleFunc("POST /tax/suggest", taxHandler.Suggest)

	protected := auth.AuthMiddleware(jwtService, protectedMux)

	s.mux.Handle("/", protected)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":  "error",
			"message": "database unavailable",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"app_env":   s.cfg.AppEnv,
		"timestamp": time.Now().UTC(),
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		w.Header().Add("Vary", "Access-Control-Request-Headers")

		if origin != "" && s.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Organization-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			if origin == "" || !s.isAllowedOrigin(origin) {
				http.Error(w, "CORS origin not allowed", http.StatusForbidden)
				return
			}

			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) isAllowedOrigin(origin string) bool {
	_, ok := s.cors[origin]
	return ok
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}