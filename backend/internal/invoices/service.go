package invoices

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"

	"github.com/rafa/fiscal-platform/backend/internal/catalog"
)

type Service struct {
	repo           *Repository
	catalogService *catalog.Service
}

func NewService(repo *Repository, catalogService *catalog.Service) *Service {
	return &Service{
		repo:           repo,
		catalogService: catalogService,
	}
}

type UploadResult struct {
	InvoiceID  string `json:"invoice_id"`
	ItemsCount int    `json:"items_count"`
}

type BatchUploadItemResult struct {
	FileName   string `json:"file_name"`
	InvoiceID  string `json:"invoice_id,omitempty"`
	ItemsCount int    `json:"items_count,omitempty"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

type BatchUploadResult struct {
	TotalFiles   int                     `json:"total_files"`
	SuccessCount int                     `json:"success_count"`
	FailedCount  int                     `json:"failed_count"`
	Results      []BatchUploadItemResult `json:"results"`
}

func (s *Service) ProcessXML(ctx context.Context, organizationID string, xmlRaw string, file io.Reader) (*UploadResult, error) {
	doc, err := ParseXML(file)
	if err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}

	recipientCNPJ := doc.NFe.InfNFe.Dest.CNPJ
	if recipientCNPJ == "" {
		recipientCNPJ = doc.NFe.InfNFe.Dest.CPF
	}

	invoiceID, err := s.repo.CreateInvoice(ctx, CreateInvoiceParams{
		OrganizationID:  organizationID,
		AccessKey:       doc.NFe.InfNFe.ID,
		Number:          doc.NFe.InfNFe.Ide.NNF,
		Series:          doc.NFe.InfNFe.Ide.Serie,
		IssuedAt:        normalizeTimestamp(doc.NFe.InfNFe.Ide.DhEmi),
		EmitterName:     doc.NFe.InfNFe.Emit.XNome,
		EmitterCNPJ:     doc.NFe.InfNFe.Emit.CNPJ,
		EmitterUF:       doc.NFe.InfNFe.Emit.Ender.UF,
		RecipientName:   doc.NFe.InfNFe.Dest.XNome,
		RecipientCNPJ:   recipientCNPJ,
		RecipientUF:     doc.NFe.InfNFe.Dest.Ender.UF,
		OperationNature: doc.NFe.InfNFe.Ide.NatOp,
		TotalAmount:     doc.NFe.InfNFe.Total.ICMSTot.VNF,
		XMLRaw:          xmlRaw,
		Status:          "processed",
	})
	if err != nil {
		return nil, err
	}

	for _, item := range doc.NFe.InfNFe.Det {
		itemNumber, _ := strconv.Atoi(item.NItem)

		icmsData := extractICMSData(item.Imposto.ICMS.InnerXML)
		ipiValue := extractIPIValue(item.Imposto.IPI)
		pisData := extractPISData(item.Imposto.PIS)
		cofinsData := extractCOFINSData(item.Imposto.COFINS)
		sourceType := inferObservedSourceType(item.Prod.CFOP)

		err := s.repo.CreateInvoiceItem(ctx, CreateInvoiceItemParams{
			InvoiceID:      invoiceID,
			ItemNumber:     itemNumber,
			ProductCode:    item.Prod.CProd,
			GTIN:           item.Prod.CEAN,
			GTINTributable: item.Prod.CEANTrib,
			Description:    item.Prod.XProd,
			NCM:            item.Prod.NCM,
			CEST:           item.Prod.CEST,
			CFOP:           item.Prod.CFOP,
			Unit:           item.Prod.UCom,
			Quantity:       item.Prod.QCom,
			UnitValue:      item.Prod.VUnCom,
			TotalValue:     item.Prod.VProd,
			ICMSCST:        icmsData.CST,
			CSOSN:          icmsData.CSOSN,
			ICMSRate:       icmsData.Rate,
			PISCST:         pisData.CST,
			PISRate:        pisData.Rate,
			COFINSCST:      cofinsData.CST,
			COFINSRate:     cofinsData.Rate,
			ICMSValue:      icmsData.Value,
			IPIValue:       ipiValue,
			PISValue:       pisData.Value,
			COFINSValue:    cofinsData.Value,
		})
		if err != nil {
			return nil, err
		}

		if s.catalogService != nil {
			_ = s.catalogService.RegisterObservedItem(ctx, catalog.RegisterObservedItemParams{
				OrganizationID:  organizationID,
				SourceInvoiceID: invoiceID,

				ProductCode: item.Prod.CProd,
				GTIN:        item.Prod.CEAN,
				Description: item.Prod.XProd,

				NCM:         item.Prod.NCM,
				CEST:        item.Prod.CEST,
				CFOP:        item.Prod.CFOP,
				PISCST:      pisData.CST,
				COFINSCST:   cofinsData.CST,
				ICMSCST:     icmsData.CST,
				CSOSN:       icmsData.CSOSN,
				ICMSValue:   icmsData.Value,
				IPIValue:    ipiValue,
				PISValue:    pisData.Value,
				COFINSValue: cofinsData.Value,
				PISRate:     pisData.Rate,
				COFINSRate:  cofinsData.Rate,
				ICMSRate:    icmsData.Rate,

				EmitterUF:       doc.NFe.InfNFe.Emit.Ender.UF,
				RecipientUF:     doc.NFe.InfNFe.Dest.Ender.UF,
				OperationNature: doc.NFe.InfNFe.Ide.NatOp,
				SourceType:      sourceType,
			})
		}
	}

	return &UploadResult{
		InvoiceID:  invoiceID,
		ItemsCount: len(doc.NFe.InfNFe.Det),
	}, nil
}

func normalizeTimestamp(v string) string {
	return v
}

func extractIPIValue(ipi IPI) string {
	if ipi.IPITrib != nil {
		return ipi.IPITrib.VIPI
	}
	return ""
}

type extractedTax struct {
	CSOSN string
	CST   string
	Rate  string
	Value string
}

func extractPISData(pis PIS) extractedTax {
	type holder struct {
		CST  string `xml:"CST"`
		PPIS string `xml:"pPIS"`
		VPIS string `xml:"vPIS"`
	}

	var item holder
	_ = xml.Unmarshal(pis.InnerXML, &item)

	if item.VPIS == "" {
		if pis.PISAliq != nil {
			item.CST = firstNonEmptyString(item.CST, pis.PISAliq.CST)
			item.PPIS = firstNonEmptyString(item.PPIS, pis.PISAliq.PPIS)
			item.VPIS = firstNonEmptyString(item.VPIS, pis.PISAliq.VPIS)
		}
		if pis.PISOutr != nil {
			item.CST = firstNonEmptyString(item.CST, pis.PISOutr.CST)
			item.PPIS = firstNonEmptyString(item.PPIS, pis.PISOutr.PPIS)
			item.VPIS = firstNonEmptyString(item.VPIS, pis.PISOutr.VPIS)
		}
	}

	return extractedTax{
		CST:   item.CST,
		Rate:  item.PPIS,
		Value: item.VPIS,
	}
}

func extractCOFINSData(cofins COFINS) extractedTax {
	type holder struct {
		CST     string `xml:"CST"`
		PCOFINS string `xml:"pCOFINS"`
		VCOFINS string `xml:"vCOFINS"`
	}

	var item holder
	_ = xml.Unmarshal(cofins.InnerXML, &item)

	if item.VCOFINS == "" {
		if cofins.COFINSAliq != nil {
			item.CST = firstNonEmptyString(item.CST, cofins.COFINSAliq.CST)
			item.PCOFINS = firstNonEmptyString(item.PCOFINS, cofins.COFINSAliq.PCOFINS)
			item.VCOFINS = firstNonEmptyString(item.VCOFINS, cofins.COFINSAliq.VCOFINS)
		}
		if cofins.COFINSOutr != nil {
			item.CST = firstNonEmptyString(item.CST, cofins.COFINSOutr.CST)
			item.PCOFINS = firstNonEmptyString(item.PCOFINS, cofins.COFINSOutr.PCOFINS)
			item.VCOFINS = firstNonEmptyString(item.VCOFINS, cofins.COFINSOutr.VCOFINS)
		}
	}

	return extractedTax{
		CST:   item.CST,
		Rate:  item.PCOFINS,
		Value: item.VCOFINS,
	}
}

func extractICMSData(innerXML []byte) extractedTax {
	type holder struct {
		CST   string `xml:"CST"`
		CSOSN string `xml:"CSOSN"`
		PICMS string `xml:"pICMS"`
		VICMS string `xml:"vICMS"`
	}

	var item holder
	_ = xml.Unmarshal(innerXML, &item)

	return extractedTax{
		CSOSN: item.CSOSN,
		CST:   item.CST,
		Rate:  item.PICMS,
		Value: item.VICMS,
	}
}

func inferObservedSourceType(cfop string) string {
	cfop = firstNonEmptyString(cfop)
	if len(cfop) > 0 {
		switch cfop[0] {
		case '1', '2', '3':
			return "invoice_import_entry"
		case '5', '6', '7':
			return "invoice_import_exit"
		}
	}

	return "invoice_import"
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func (s *Service) ListInvoices(ctx context.Context, organizationID string) ([]InvoiceListItem, error) {
	return s.repo.ListInvoices(ctx, organizationID)
}

func (s *Service) GetInvoiceByID(ctx context.Context, organizationID, invoiceID string) (*InvoiceDetail, error) {
	return s.repo.GetInvoiceByID(ctx, organizationID, invoiceID)
}
