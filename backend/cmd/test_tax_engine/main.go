package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine"
	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

func main() {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	dbpool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("create db pool: %v", err)
	}
	defer dbpool.Close()

	engine, err := taxengine.New(taxengine.Dependencies{
		DB: dbpool,
	})
	if err != nil {
		log.Fatalf("build tax engine: %v", err)
	}

	input := domain.EvaluateInput{
		TenantID:       "tenant-dev",
		OrganizationID: "org-dev",
		InvoiceID:      stringPtr("invoice-test-001"),
		InvoiceItemID:  stringPtr("invoice-item-test-001"),

		Issuer: domain.PartyContext{
			CNPJ:              "12345678000199",
			UF:                "GO",
			CRT:               "3",
			IE:                "109876543",
			IsContributorICMS: true,
			IsFinalConsumer:   false,
			TaxRegimeCode:     "3",
			MunicipalityCode:  "5208707",
			CountryCode:       "1058",
		},

		Recipient: domain.PartyContext{
			CNPJ:              "99887766000155",
			UF:                "GO",
			CRT:               "3",
			IE:                "112233445",
			IsContributorICMS: true,
			IsFinalConsumer:   false,
			TaxRegimeCode:     "3",
			MunicipalityCode:  "5208707",
			CountryCode:       "1058",
		},

		Operation: domain.OperationContext{
			DocumentType:          domain.DocumentTypeNFe,
			OperationType:         domain.OperationTypeExit,
			OperationScope:        domain.OperationScopeInternal,
			CFOP:                  "5102",
			FinNFe:                "1",
			PresenceIndicator:     "1",
			PurposeCode:           "1",
			IsReturn:              false,
			IsTransfer:            false,
			IsResale:              true,
			IsImport:              false,
			IsExport:              false,
			HasInterstateDelivery: false,
		},

		Product: domain.ProductContext{
			ItemNumber:            1,
			SupplierID:            "supplier-test-001",
			SupplierProductCode:   "SKU-001",
			Description:           "Cerveja Pilsen Lata 350ml",
			AdditionalDescription: "Produto teste da engine",
			GTIN:                  "7891234567895",
			NCM:                   "22030000",
			EXTIPI:                "",
			CEST:                  "",
			OriginCode:            "0",
			Unit:                  "UN",
			CommercialUnit:        "UN",
			TributaryUnit:         "UN",
			Quantity:              10,
		},

		Values: domain.ValueContext{
			UnitValue:          5.50,
			GrossValue:         55.00,
			DiscountValue:      0,
			FreightValue:       0,
			InsuranceValue:     0,
			OtherExpensesValue: 0,
			IPIValue:           0,
			ICMSBaseValue:      0,
			TotalValue:         55.00,
		},

		Metadata: domain.MetadataContext{
			InvoiceNumber:  "12345",
			InvoiceSeries:  "1",
			InvoiceKey:     "52123456780001995550010000123451000012345",
			IssueDate:      time.Now(),
			ImportedAt:     time.Now(),
			Source:         "manual_test",
			SourceFileName: "manual_test.xml",
			UserID:         "user-test-001",
			RequestID:      "request-test-001",
		},
	}

	output, err := engine.Evaluate(ctx, input)
	if err != nil {
		log.Fatalf("evaluate tax engine: %v", err)
	}

	pretty, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("marshal output: %v", err)
	}

	fmt.Println(string(pretty))
}

func stringPtr(v string) *string {
	return &v
}