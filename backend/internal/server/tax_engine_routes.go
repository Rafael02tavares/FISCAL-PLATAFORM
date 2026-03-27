package server

import (
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafa/fiscal-platform/backend/internal/invoices"
)

type TaxEngineRoutesResult struct {
	Module *invoices.TaxEngineModule
}

func RegisterTaxEngineRoutes(
	mux *http.ServeMux,
	db *pgxpool.Pool,
) (*TaxEngineRoutesResult, error) {
	if mux == nil {
		return nil, fmt.Errorf("register tax engine routes: mux is required")
	}
	if db == nil {
		return nil, fmt.Errorf("register tax engine routes: db is required")
	}

	module, err := invoices.NewTaxEngineModule(invoices.TaxEngineModuleDependencies{
		DB: db,
	})
	if err != nil {
		return nil, fmt.Errorf("register tax engine routes: create tax engine module: %w", err)
	}

	mux.HandleFunc("POST /invoices/process-taxes", module.Handler.HandleProcessInvoiceTaxes)

	return &TaxEngineRoutesResult{
		Module: module,
	}, nil
}