package invoices

import (
	"encoding/xml"
	"io"
)

type NFeProc struct {
	XMLName xml.Name `xml:"nfeProc"`
	NFe     NFeNode  `xml:"NFe"`
}

type NFeNode struct {
	InfNFe InfNFe `xml:"infNFe"`
}

type InfNFe struct {
	ID    string    `xml:"Id,attr"`
	Ide   Ide       `xml:"ide"`
	Emit  Emit      `xml:"emit"`
	Dest  Dest      `xml:"dest"`
	Total Total     `xml:"total"`
	Det   []DetItem `xml:"det"`
}

type Ide struct {
	NNF   string `xml:"nNF"`
	Serie string `xml:"serie"`
	DhEmi string `xml:"dhEmi"`
	NatOp string `xml:"natOp"`
}

type Emit struct {
	XNome string    `xml:"xNome"`
	CNPJ  string    `xml:"CNPJ"`
	Ender EnderBase `xml:"enderEmit"`
}

type Dest struct {
	XNome string    `xml:"xNome"`
	CNPJ  string    `xml:"CNPJ"`
	CPF   string    `xml:"CPF"`
	Ender EnderBase `xml:"enderDest"`
}

type EnderBase struct {
	UF string `xml:"UF"`
}

type Total struct {
	ICMSTot ICMSTot `xml:"ICMSTot"`
}

type ICMSTot struct {
	VNF string `xml:"vNF"`
}

type DetItem struct {
	NItem   string  `xml:"nItem,attr"`
	Prod    Prod    `xml:"prod"`
	Imposto Imposto `xml:"imposto"`
}

type Prod struct {
	CProd    string `xml:"cProd"`
	CEAN     string `xml:"cEAN"`
	CEANTrib string `xml:"cEANTrib"`
	XProd    string `xml:"xProd"`
	NCM      string `xml:"NCM"`
	CEST     string `xml:"CEST"`
	CFOP     string `xml:"CFOP"`
	UCom     string `xml:"uCom"`
	QCom     string `xml:"qCom"`
	VUnCom   string `xml:"vUnCom"`
	VProd    string `xml:"vProd"`
}

type Imposto struct {
	IPI    IPI    `xml:"IPI"`
	PIS    PIS    `xml:"PIS"`
	COFINS COFINS `xml:"COFINS"`
	ICMS   ICMS   `xml:"ICMS"`
}

type IPI struct {
	IPITrib *IPITrib `xml:"IPITrib"`
}

type IPITrib struct {
	VIPI string `xml:"vIPI"`
}

type PIS struct {
	PISAliq *PISAliq `xml:"PISAliq"`
	PISOutr *PISOutr `xml:"PISOutr"`
}

type PISAliq struct {
	VPIS string `xml:"vPIS"`
}

type PISOutr struct {
	VPIS string `xml:"vPIS"`
}

type COFINS struct {
	COFINSAliq *COFINSAliq `xml:"COFINSAliq"`
	COFINSOutr *COFINSOutr `xml:"COFINSOutr"`
}

type COFINSAliq struct {
	VCOFINS string `xml:"vCOFINS"`
}

type COFINSOutr struct {
	VCOFINS string `xml:"vCOFINS"`
}

type ICMS struct {
	InnerXML []byte `xml:",innerxml"`
}

func ParseXML(r io.Reader) (*NFeProc, error) {
	var doc NFeProc
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}
