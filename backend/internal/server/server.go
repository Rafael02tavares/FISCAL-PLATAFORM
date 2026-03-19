package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafa/fiscal-platform/backend/internal/auth"
	"github.com/rafa/fiscal-platform/backend/internal/catalog"
	"github.com/rafa/fiscal-platform/backend/internal/companies"
	"github.com/rafa/fiscal-platform/backend/internal/config"
	"github.com/rafa/fiscal-platform/backend/internal/fiscaloperations"
	"github.com/rafa/fiscal-platform/backend/internal/invoices"
	"github.com/rafa/fiscal-platform/backend/internal/legalbasis"
	"github.com/rafa/fiscal-platform/backend/internal/ncm"
	"github.com/rafa/fiscal-platform/backend/internal/organizations"
	"github.com/rafa/fiscal-platform/backend/internal/tax"
)

type Server struct {
	cfg config.Config
	db  *pgxpool.Pool
	mux *http.ServeMux
}

func New(cfg config.Config, db *pgxpool.Pool) http.Handler {
	s := &Server{
		cfg: cfg,
		db:  db,
		mux: http.NewServeMux(),
	}

	s.registerRoutes()

	return withCORS(s.mux)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// auth
	authRepo := auth.NewRepository(s.db)
	authService := auth.NewService(authRepo)
	jwtService := auth.NewJWT(s.cfg.JWTSecret)
	authHandler := auth.NewHandler(authService, jwtService)

	s.mux.HandleFunc("POST /auth/register", authHandler.Register)
	s.mux.HandleFunc("POST /auth/login", authHandler.Login)

	// consulta pública CNPJ
	companyClient := companies.NewClient()
	companyService := companies.NewService(companyClient)
	companyHandler := companies.NewHandler(companyService)
	s.mux.HandleFunc("GET /companies/lookup", companyHandler.Lookup)

	// operações fiscais
	fiscalOpRepo := fiscaloperations.NewRepository(s.db)
	fiscalOpService := fiscaloperations.NewService(fiscalOpRepo)
	fiscalOpHandler := fiscaloperations.NewHandler(fiscalOpService)
	s.mux.HandleFunc("GET /fiscal-operations", fiscalOpHandler.List)

	// catálogo NCM
	ncmRepo := ncm.NewRepository(s.db)
	ncmService := ncm.NewService(ncmRepo)
	ncmHandler := ncm.NewHandler(ncmService)
	s.mux.HandleFunc("GET /ncm", ncmHandler.List)
	s.mux.HandleFunc("GET /ncm/find", ncmHandler.GetByCode)
	s.mux.HandleFunc("GET /ncm/search", ncmHandler.Search)

	// base legal
	legalRepo := legalbasis.NewRepository(s.db)
	legalService := legalbasis.NewService(legalRepo)
	legalHandler := legalbasis.NewHandler(legalService)
	s.mux.HandleFunc("GET /legal-sources", legalHandler.ListLegalSources)
	s.mux.HandleFunc("POST /legal-sources", legalHandler.CreateLegalSource)
	s.mux.HandleFunc("GET /legal-rules", legalHandler.ListLegalRuleMappings)
	s.mux.HandleFunc("POST /legal-rules", legalHandler.CreateLegalRuleMapping)

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("GET /auth/me", authHandler.Me)

	// organizations
	orgRepo := organizations.NewRepository(s.db)
	orgService := organizations.NewService(orgRepo)
	orgHandler := organizations.NewHandler(orgService)

	protectedMux.HandleFunc("POST /organizations", orgHandler.Create)
	protectedMux.HandleFunc("GET /organizations", orgHandler.List)

	// catálogo / invoices
	catalogRepo := catalog.NewRepository(s.db)
	catalogService := catalog.NewService(catalogRepo)

	invoiceRepo := invoices.NewRepository(s.db)
	invoiceService := invoices.NewService(invoiceRepo, catalogService)
	invoiceHandler := invoices.NewHandler(invoiceService, orgService)

	protectedMux.HandleFunc("POST /invoices/upload", invoiceHandler.Upload)
	protectedMux.HandleFunc("GET /invoices", invoiceHandler.List)
	protectedMux.HandleFunc("GET /invoices/", invoiceHandler.GetByID)

	// tax engine
	taxRepo := tax.NewRepository(s.db)
	taxService := tax.NewService(taxRepo, fiscalOpService, legalService)
	taxHandler := tax.NewHandler(taxService, orgService)

	protectedMux.HandleFunc("POST /tax/suggest", taxHandler.Suggest)

	protected := auth.AuthMiddleware(jwtService, protectedMux)

	s.mux.Handle("/auth/me", protected)
	s.mux.Handle("/organizations", protected)
	s.mux.Handle("/invoices", protected)
	s.mux.Handle("/invoices/", protected)
	s.mux.Handle("/tax/suggest", protected)
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

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4321")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Organization-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
