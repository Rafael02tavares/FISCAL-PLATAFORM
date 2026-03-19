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

		icmsValue := extractICMSValue(item.Imposto.ICMS.InnerXML)
		ipiValue := extractIPIValue(item.Imposto.IPI)
		pisValue := extractPISValue(item.Imposto.PIS)
		cofinsValue := extractCOFINSValue(item.Imposto.COFINS)

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
			ICMSValue:      icmsValue,
			IPIValue:       ipiValue,
			PISValue:       pisValue,
			COFINSValue:    cofinsValue,
		})
		if err != nil {
			return nil, err
		}

		if s.catalogService != nil {
			_ = s.catalogService.RegisterObservedItem(ctx, catalog.RegisterObservedItemParams{
				OrganizationID:  organizationID,
				SourceInvoiceID: invoiceID,

				GTIN:        item.Prod.CEAN,
				Description: item.Prod.XProd,

				NCM:         item.Prod.NCM,
				CEST:        item.Prod.CEST,
				CFOP:        item.Prod.CFOP,
				ICMSValue:   icmsValue,
				IPIValue:    ipiValue,
				PISValue:    pisValue,
				COFINSValue: cofinsValue,

				EmitterUF:       doc.NFe.InfNFe.Emit.Ender.UF,
				RecipientUF:     doc.NFe.InfNFe.Dest.Ender.UF,
				OperationNature: doc.NFe.InfNFe.Ide.NatOp,
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

func extractPISValue(pis PIS) string {
	if pis.PISAliq != nil {
		return pis.PISAliq.VPIS
	}
	if pis.PISOutr != nil {
		return pis.PISOutr.VPIS
	}
	return ""
}

func extractCOFINSValue(cofins COFINS) string {
	if cofins.COFINSAliq != nil {
		return cofins.COFINSAliq.VCOFINS
	}
	if cofins.COFINSOutr != nil {
		return cofins.COFINSOutr.VCOFINS
	}
	return ""
}

func extractICMSValue(innerXML []byte) string {
	type valueHolder struct {
		VICMS string `xml:"vICMS"`
	}

	var holder valueHolder
	_ = xml.Unmarshal(innerXML, &holder)

	return holder.VICMS
}

func (s *Service) ListInvoices(ctx context.Context, organizationID string) ([]InvoiceListItem, error) {
	return s.repo.ListInvoices(ctx, organizationID)
}

func (s *Service) GetInvoiceByID(ctx context.Context, organizationID, invoiceID string) (*InvoiceDetail, error) {
	return s.repo.GetInvoiceByID(ctx, organizationID, invoiceID)
}
