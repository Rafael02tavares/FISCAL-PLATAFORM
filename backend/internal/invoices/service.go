package invoices

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Rafael02tavares/FISCAL-PLATAFORM/backend/internal/catalog"
)

var (
	ErrInvalidInvoiceData = errors.New("invalid invoice data")
	ErrInvoiceNotFound    = errors.New("invoice not found")
)

const processedInvoiceStatus = "processed"

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
	organizationID = strings.TrimSpace(organizationID)
	xmlRaw = strings.TrimSpace(xmlRaw)

	if organizationID == "" || file == nil || xmlRaw == "" {
		return nil, ErrInvalidInvoiceData
	}

	doc, err := ParseXML(file)
	if err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}

	recipientCNPJ := strings.TrimSpace(doc.NFe.InfNFe.Dest.CNPJ)
	if recipientCNPJ == "" {
		recipientCNPJ = strings.TrimSpace(doc.NFe.InfNFe.Dest.CPF)
	}

	accessKey := strings.TrimPrefix(strings.TrimSpace(doc.NFe.InfNFe.ID), "NFe")

	invoiceID, err := s.repo.CreateInvoice(ctx, CreateInvoiceParams{
		OrganizationID:  organizationID,
		AccessKey:       accessKey,
		Number:          strings.TrimSpace(doc.NFe.InfNFe.Ide.NNF),
		Series:          strings.TrimSpace(doc.NFe.InfNFe.Ide.Serie),
		IssuedAt:        normalizeTimestamp(doc.NFe.InfNFe.Ide.DhEmi),
		EmitterName:     strings.TrimSpace(doc.NFe.InfNFe.Emit.XNome),
		EmitterCNPJ:     strings.TrimSpace(doc.NFe.InfNFe.Emit.CNPJ),
		EmitterUF:       strings.ToUpper(strings.TrimSpace(doc.NFe.InfNFe.Emit.Ender.UF)),
		RecipientName:   strings.TrimSpace(doc.NFe.InfNFe.Dest.XNome),
		RecipientCNPJ:   recipientCNPJ,
		RecipientUF:     strings.ToUpper(strings.TrimSpace(doc.NFe.InfNFe.Dest.Ender.UF)),
		OperationNature: strings.TrimSpace(doc.NFe.InfNFe.Ide.NatOp),
		TotalAmount:     strings.TrimSpace(doc.NFe.InfNFe.Total.ICMSTot.VNF),
		XMLRaw:          xmlRaw,
		Status:          processedInvoiceStatus,
	})
	if err != nil {
		return nil, err
	}

	for _, item := range doc.NFe.InfNFe.Det {
		itemNumber, _ := strconv.Atoi(strings.TrimSpace(item.NItem))

		icmsValue := strings.TrimSpace(extractICMSValue(item.Imposto.ICMS.InnerXML))
		ipiValue := strings.TrimSpace(extractIPIValue(item.Imposto.IPI))
		pisValue := strings.TrimSpace(extractPISValue(item.Imposto.PIS))
		cofinsValue := strings.TrimSpace(extractCOFINSValue(item.Imposto.COFINS))

		err := s.repo.CreateInvoiceItem(ctx, CreateInvoiceItemParams{
			InvoiceID:      invoiceID,
			ItemNumber:     itemNumber,
			ProductCode:    strings.TrimSpace(item.Prod.CProd),
			GTIN:           normalizeGTIN(item.Prod.CEAN),
			GTINTributable: normalizeGTIN(item.Prod.CEANTrib),
			Description:    strings.TrimSpace(item.Prod.XProd),
			NCM:            onlyDigits(item.Prod.NCM),
			CEST:           onlyDigits(item.Prod.CEST),
			CFOP:           onlyDigits(item.Prod.CFOP),
			Unit:           strings.TrimSpace(item.Prod.UCom),
			Quantity:       strings.TrimSpace(item.Prod.QCom),
			UnitValue:      strings.TrimSpace(item.Prod.VUnCom),
			TotalValue:     strings.TrimSpace(item.Prod.VProd),
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
				GTIN:            normalizeGTIN(item.Prod.CEAN),
				Description:     strings.TrimSpace(item.Prod.XProd),
				NCM:             onlyDigits(item.Prod.NCM),
				CEST:            onlyDigits(item.Prod.CEST),
				CFOP:            onlyDigits(item.Prod.CFOP),
				ICMSValue:       icmsValue,
				IPIValue:        ipiValue,
				PISValue:        pisValue,
				COFINSValue:     cofinsValue,
				EmitterUF:       strings.ToUpper(strings.TrimSpace(doc.NFe.InfNFe.Emit.Ender.UF)),
				RecipientUF:     strings.ToUpper(strings.TrimSpace(doc.NFe.InfNFe.Dest.Ender.UF)),
				OperationNature: strings.TrimSpace(doc.NFe.InfNFe.Ide.NatOp),
			})
		}
	}

	return &UploadResult{
		InvoiceID:  invoiceID,
		ItemsCount: len(doc.NFe.InfNFe.Det),
	}, nil
}

func (s *Service) ListInvoices(ctx context.Context, organizationID string) ([]InvoiceListItem, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, ErrInvalidInvoiceData
	}

	return s.repo.ListInvoices(ctx, organizationID)
}

func (s *Service) GetInvoiceByID(ctx context.Context, organizationID, invoiceID string) (*InvoiceDetail, error) {
	organizationID = strings.TrimSpace(organizationID)
	invoiceID = strings.TrimSpace(invoiceID)

	if organizationID == "" || invoiceID == "" {
		return nil, ErrInvalidInvoiceData
	}

	invoice, err := s.repo.GetInvoiceByID(ctx, organizationID, invoiceID)
	if err != nil {
		if isNotFoundError(err) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	if invoice == nil {
		return nil, ErrInvoiceNotFound
	}

	return invoice, nil
}

func normalizeTimestamp(v string) string {
	return strings.TrimSpace(v)
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

func normalizeGTIN(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "SEM GTIN" {
		return ""
	}
	return value
}

func onlyDigits(value string) string {
	var b strings.Builder
	b.Grow(len(value))

	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}

	return b.String()
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no rows")
}