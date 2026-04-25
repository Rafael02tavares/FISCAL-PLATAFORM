export type ReformCSTInfo = {
  code: string;
  label: string;
  indicators: string[];
};

export type CClassTribInfo = {
  code: string;
  label: string;
  cst: string;
};

export type ReformLegalReference = {
  code: string;
  title: string;
  source: string;
  summary: string;
  url: string;
};

const REFORM_CST: Record<string, ReformCSTInfo> = {
  "000": { code: "000", label: "Tributacao integral", indicators: ["exige tributacao"] },
  "010": { code: "010", label: "Tributacao com aliquotas uniformes", indicators: ["aliquota uniforme"] },
  "011": { code: "011", label: "Tributacao com aliquotas uniformes reduzidas", indicators: ["aliquota uniforme", "reducao de aliquota"] },
  "200": { code: "200", label: "Aliquota reduzida", indicators: ["reducao de aliquota"] },
  "220": { code: "220", label: "Aliquota fixa", indicators: ["aliquota fixa"] },
  "221": { code: "221", label: "Aliquota fixa proporcional", indicators: ["aliquota fixa proporcional"] },
  "222": { code: "222", label: "Reducao de Base de Calculo", indicators: ["reducao de BC"] },
  "400": { code: "400", label: "Isencao", indicators: ["sem aliquota"] },
  "410": { code: "410", label: "Imunidade e nao incidencia", indicators: ["sem aliquota"] },
  "510": { code: "510", label: "Diferimento", indicators: ["diferimento"] },
  "515": { code: "515", label: "Diferimento com reducao de aliquota", indicators: ["diferimento", "reducao de aliquota"] },
  "550": { code: "550", label: "Suspensao", indicators: ["suspensao"] },
  "620": { code: "620", label: "Tributacao Monofasica", indicators: ["monofasica"] },
  "800": { code: "800", label: "Transferencia de credito", indicators: ["transferencia de credito"] },
  "810": { code: "810", label: "Ajuste de IBS na ZFM", indicators: ["IBS ZFM"] },
  "811": { code: "811", label: "Ajustes", indicators: ["ajuste de competencia"] },
  "820": { code: "820", label: "Tributacao em documento especifico", indicators: ["documento especifico"] },
  "830": { code: "830", label: "Exclusao da Base de Calculo", indicators: ["exclusao de BC"] },
};

const CCLASS_TRIB_EXAMPLES: Record<string, CClassTribInfo> = {
  "000001": { code: "000001", cst: "000", label: "Situacoes tributadas integralmente pelo IBS e CBS" },
  "200003": { code: "200003", cst: "200", label: "Vendas de produtos destinados a alimentacao humana" },
  "200014": { code: "200014", cst: "200", label: "Produtos horticolas, frutas e ovos" },
  "200034": { code: "200034", cst: "200", label: "Alimentos destinados ao consumo humano" },
  "400001": { code: "400001", cst: "400", label: "Transporte publico coletivo de passageiros" },
  "410004": { code: "410004", cst: "410", label: "Exportacoes de bens e servicos" },
  "410008": { code: "410008", cst: "410", label: "Livros, jornais, periodicos e papel destinado a impressao" },
  "410029": { code: "410029", cst: "410", label: "Operacoes acobertadas somente pelo ICMS" },
  "510001": { code: "510001", cst: "510", label: "Diferimento com energia eletrica" },
  "550001": { code: "550001", cst: "550", label: "Exportacoes de bens materiais" },
  "620001": { code: "620001", cst: "620", label: "Tributacao monofasica sobre combustiveis" },
};

export function getReformCSTInfo(value: unknown): ReformCSTInfo | null {
  const code = String(value || "").replace(/\D/g, "").slice(0, 3).padStart(3, "0");
  return REFORM_CST[code] || null;
}

export function getCClassTribInfo(value: unknown): CClassTribInfo | null {
  const code = String(value || "").replace(/\D/g, "").padStart(6, "0");
  return CCLASS_TRIB_EXAMPLES[code] || null;
}

export const reformCSTReference = Object.values(REFORM_CST);
export const cClassTribExamples = Object.values(CCLASS_TRIB_EXAMPLES);

export const reformLegalReferences: ReformLegalReference[] = [
  {
    code: "LC214_ART4",
    title: "Hipotese geral de incidencia do IBS e da CBS",
    source: "Lei Complementar 214/2025, art. 4",
    summary:
      "IBS e CBS incidem sobre operacoes onerosas com bens ou servicos; operacoes nao onerosas somente nas hipoteses expressas da lei.",
    url: "https://www.planalto.gov.br/ccivil_03/leis/lcp/lcp214.htm#art4",
  },
  {
    code: "LC214_ART4_PAR2",
    title: "Operacao onerosa com contraprestacao",
    source: "Lei Complementar 214/2025, art. 4, § 2",
    summary:
      "Inclui compra e venda, troca, permuta, locacao, licenciamento, cessao, mutuo oneroso, instituicao onerosa de direitos reais, arrendamento e prestacao de servicos.",
    url: "https://www.planalto.gov.br/ccivil_03/leis/lcp/lcp214.htm#art4",
  },
];
