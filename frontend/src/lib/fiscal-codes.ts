export type FiscalCodeInfo = {
  code: string;
  label: string;
  group: "normal" | "st" | "exempt" | "other";
  note: string;
};

export type PISCOFINSRegimeAdvice = {
  status: "ok" | "attention" | "warning";
  title: string;
  detail: string;
};

export type ICMSRegimeAdvice = {
  status: "ok" | "attention" | "warning";
  title: string;
  detail: string;
};

const ICMS_CST: Record<string, FiscalCodeInfo> = {
  "00": { code: "00", label: "Tributada integralmente", group: "normal", note: "Operacao tributada pelo ICMS sem reducao ou ST." },
  "10": { code: "10", label: "Tributada com cobranca de ICMS ST", group: "st", note: "Indica ICMS proprio e cobranca por substituicao tributaria." },
  "20": { code: "20", label: "Com reducao de base de calculo", group: "normal", note: "Exige percentual de reducao e fundamento aplicavel." },
  "30": { code: "30", label: "Isenta/nao tributada com ICMS ST", group: "st", note: "Ha tratamento sem ICMS proprio e cobranca por ST." },
  "40": { code: "40", label: "Isenta", group: "exempt", note: "Exige fundamento legal de isencao." },
  "41": { code: "41", label: "Nao tributada", group: "exempt", note: "Operacao fora da tributacao de ICMS no contexto informado." },
  "50": { code: "50", label: "Suspensao", group: "exempt", note: "Tributacao suspensa conforme regra legal especifica." },
  "51": { code: "51", label: "Diferimento", group: "normal", note: "ICMS postergado para etapa posterior." },
  "60": { code: "60", label: "ICMS cobrado anteriormente por ST", group: "st", note: "Mercadoria em condicao de substituido tributario; normalmente preserva CFOP ST como 5405 em venda interna." },
  "70": { code: "70", label: "Reducao de base com ICMS ST", group: "st", note: "Combina reducao de base com substituicao tributaria." },
  "90": { code: "90", label: "Outras", group: "other", note: "Exige regra fiscal especifica para evitar classificacao generica." },
};

const ICMS_ORIGIN: Record<string, string> = {
  "0": "Nacional",
  "1": "Estrangeira - importacao direta",
  "2": "Estrangeira - adquirida no mercado interno",
  "3": "Nacional com conteudo de importacao superior a 40% e ate 70%",
  "4": "Nacional conforme processo produtivo basico",
  "5": "Nacional com conteudo de importacao inferior ou igual a 40%",
  "6": "Estrangeira - importacao direta sem similar nacional",
  "7": "Estrangeira - adquirida no mercado interno sem similar nacional",
  "8": "Nacional com conteudo de importacao superior a 70%",
};

const CSOSN: Record<string, FiscalCodeInfo> = {
  "101": { code: "101", label: "Tributada com permissao de credito", group: "normal", note: "Usado por optante do Simples Nacional com permissao de credito." },
  "102": { code: "102", label: "Tributada sem permissao de credito", group: "normal", note: "Operacao tributada no Simples sem destaque de credito." },
  "103": { code: "103", label: "Isencao para faixa de receita", group: "exempt", note: "Exige enquadramento de faixa de receita bruta." },
  "201": { code: "201", label: "Tributada com credito e ICMS ST", group: "st", note: "Simples Nacional com permissao de credito e cobranca por ST." },
  "202": { code: "202", label: "Tributada sem credito e ICMS ST", group: "st", note: "Simples Nacional sem permissao de credito e com ST." },
  "203": { code: "203", label: "Isencao para faixa de receita e ICMS ST", group: "st", note: "Combina isencao do Simples e cobranca por ST." },
  "300": { code: "300", label: "Imune", group: "exempt", note: "Operacao imune no Simples Nacional." },
  "400": { code: "400", label: "Nao tributada pelo Simples", group: "exempt", note: "Operacao nao sujeita ao ICMS dentro do Simples." },
  "500": { code: "500", label: "ICMS cobrado anteriormente por ST/antecipacao", group: "st", note: "Equivalente operacional ao CST 60 para Simples Nacional." },
  "900": { code: "900", label: "Outros", group: "other", note: "Exige regra especifica para fechamento do tratamento." },
};

const PIS_COFINS_CST: Record<string, FiscalCodeInfo> = {
  "01": { code: "01", label: "Operacao tributavel com aliquota basica", group: "normal", note: "Saida tributada pela aliquota basica do regime aplicavel." },
  "02": { code: "02", label: "Operacao tributavel com aliquota diferenciada", group: "normal", note: "Saida tributada com aliquota diferenciada; exige regra por produto/regime." },
  "03": { code: "03", label: "Operacao tributavel por unidade de medida", group: "normal", note: "Tributacao por quantidade/unidade de medida." },
  "04": { code: "04", label: "Monofasica - revenda a aliquota zero", group: "other", note: "Usado em cadeias monofasicas na revenda com aliquota zero." },
  "05": { code: "05", label: "Operacao tributavel por substituicao tributaria", group: "st", note: "Indica tratamento de substituicao tributaria para PIS/COFINS." },
  "06": { code: "06", label: "Operacao tributavel a aliquota zero", group: "exempt", note: "Saida tributavel, mas com aliquota zero." },
  "07": { code: "07", label: "Operacao isenta da contribuicao", group: "exempt", note: "Exige fundamento de isencao." },
  "08": { code: "08", label: "Operacao sem incidencia da contribuicao", group: "exempt", note: "Operacao fora da incidencia da contribuicao." },
  "09": { code: "09", label: "Operacao com suspensao da contribuicao", group: "exempt", note: "Tributacao suspensa conforme regra aplicavel." },
  "49": { code: "49", label: "Outras operacoes de saida", group: "other", note: "Classificacao generica de saida; recomenda regra especifica." },
  "50": { code: "50", label: "Credito - receita tributada no mercado interno", group: "normal", note: "Entrada com direito a credito vinculado a receita tributada." },
  "51": { code: "51", label: "Credito - receita nao tributada no mercado interno", group: "normal", note: "Entrada com credito vinculado a receita nao tributada." },
  "52": { code: "52", label: "Credito - receita de exportacao", group: "normal", note: "Entrada com direito a credito vinculado a exportacao." },
  "53": { code: "53", label: "Credito - receitas tributadas e nao tributadas", group: "normal", note: "Entrada com credito para receitas mistas no mercado interno." },
  "54": { code: "54", label: "Credito - mercado interno e exportacao", group: "normal", note: "Entrada com credito para receita tributada e exportacao." },
  "55": { code: "55", label: "Credito - nao tributadas e exportacao", group: "normal", note: "Entrada com credito para receita nao tributada e exportacao." },
  "56": { code: "56", label: "Credito - receitas mistas e exportacao", group: "normal", note: "Entrada com credito para receitas tributadas, nao tributadas e exportacao." },
  "60": { code: "60", label: "Credito presumido - receita tributada", group: "normal", note: "Aquisicao com credito presumido vinculado a receita tributada." },
  "61": { code: "61", label: "Credito presumido - receita nao tributada", group: "normal", note: "Aquisicao com credito presumido vinculado a receita nao tributada." },
  "62": { code: "62", label: "Credito presumido - exportacao", group: "normal", note: "Credito presumido vinculado a receita de exportacao." },
  "63": { code: "63", label: "Credito presumido - receitas mistas", group: "normal", note: "Credito presumido vinculado a receitas tributadas e nao tributadas." },
  "64": { code: "64", label: "Credito presumido - mercado interno/exportacao", group: "normal", note: "Credito presumido para receita tributada e exportacao." },
  "65": { code: "65", label: "Credito presumido - nao tributada/exportacao", group: "normal", note: "Credito presumido para receita nao tributada e exportacao." },
  "66": { code: "66", label: "Credito presumido - receitas mistas/exportacao", group: "normal", note: "Credito presumido para receitas mistas e exportacao." },
  "67": { code: "67", label: "Credito presumido - outras operacoes", group: "other", note: "Credito presumido em outras hipoteses." },
  "70": { code: "70", label: "Aquisicao sem direito a credito", group: "other", note: "Entrada sem apropriacao de credito de PIS/COFINS." },
  "71": { code: "71", label: "Aquisicao com isencao", group: "exempt", note: "Entrada com isencao." },
  "72": { code: "72", label: "Aquisicao com suspensao", group: "exempt", note: "Entrada com suspensao da contribuicao." },
  "73": { code: "73", label: "Aquisicao a aliquota zero", group: "exempt", note: "Entrada sujeita a aliquota zero." },
  "74": { code: "74", label: "Aquisicao sem incidencia", group: "exempt", note: "Entrada sem incidencia da contribuicao." },
  "75": { code: "75", label: "Aquisicao por substituicao tributaria", group: "st", note: "Entrada sujeita a substituicao tributaria de PIS/COFINS." },
  "98": { code: "98", label: "Outras operacoes de entrada", group: "other", note: "Classificacao generica de entrada." },
  "99": { code: "99", label: "Outras operacoes", group: "other", note: "Classificacao residual; recomenda regra especifica." },
};

const IPI_CST: Record<string, FiscalCodeInfo> = {
  "00": { code: "00", label: "Entrada com recuperacao de credito", group: "normal", note: "Entrada permite recuperacao de credito de IPI." },
  "01": { code: "01", label: "Entrada tributavel com aliquota zero", group: "exempt", note: "Entrada tributavel, mas com aliquota zero." },
  "02": { code: "02", label: "Entrada isenta", group: "exempt", note: "Entrada com isencao de IPI." },
  "03": { code: "03", label: "Entrada nao tributada", group: "exempt", note: "Entrada nao tributada pelo IPI." },
  "04": { code: "04", label: "Entrada imune", group: "exempt", note: "Entrada com imunidade de IPI." },
  "05": { code: "05", label: "Entrada com suspensao", group: "exempt", note: "Entrada com suspensao de IPI." },
  "49": { code: "49", label: "Outras entradas", group: "other", note: "Classificacao residual para entradas de IPI." },
  "50": { code: "50", label: "Saida tributada", group: "normal", note: "Saida tributada pelo IPI." },
  "51": { code: "51", label: "Saida tributavel com aliquota zero", group: "exempt", note: "Saida tributavel, mas com aliquota zero." },
  "52": { code: "52", label: "Saida isenta", group: "exempt", note: "Saida com isencao de IPI." },
  "53": { code: "53", label: "Saida nao tributada", group: "exempt", note: "Saida nao tributada pelo IPI." },
  "54": { code: "54", label: "Saida imune", group: "exempt", note: "Saida com imunidade de IPI." },
  "55": { code: "55", label: "Saida com suspensao", group: "exempt", note: "Saida com suspensao de IPI." },
  "99": { code: "99", label: "Outras saidas", group: "other", note: "Classificacao residual para saidas de IPI." },
};

export function getICMSCSTInfo(value: unknown): FiscalCodeInfo | null {
  const code = String(value || "").replace(/\D/g, "").slice(-2).padStart(2, "0");
  return ICMS_CST[code] || null;
}

export function getICMSOriginInfo(value: unknown): string {
  const digits = String(value || "").replace(/\D/g, "");
  if (digits.length < 3) return "";
  return ICMS_ORIGIN[digits.slice(0, 1)] || "";
}

export function getCSOSNInfo(value: unknown): FiscalCodeInfo | null {
  const code = String(value || "").replace(/\D/g, "");
  return CSOSN[code] || null;
}

export function isSTFiscalCode(value: unknown): boolean {
  const icms = getICMSCSTInfo(value);
  const csosn = getCSOSNInfo(value);
  const pisCofins = getPISCOFINSCSTInfo(value);
  return icms?.group === "st" || csosn?.group === "st" || pisCofins?.group === "st";
}

export function getPISCOFINSCSTInfo(value: unknown): FiscalCodeInfo | null {
  const code = String(value || "").replace(/\D/g, "").padStart(2, "0");
  return PIS_COFINS_CST[code] || null;
}

export function getIPICSTInfo(value: unknown): FiscalCodeInfo | null {
  const code = String(value || "").replace(/\D/g, "").padStart(2, "0");
  return IPI_CST[code] || null;
}

export function getPISCOFINSRegimeAdvice(regime: unknown, pisCST: unknown, cofinsCST: unknown): PISCOFINSRegimeAdvice {
  const normalizedRegime = String(regime || "")
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase();
  const pis = String(pisCST || "").replace(/\D/g, "").padStart(2, "0");
  const cofins = String(cofinsCST || "").replace(/\D/g, "").padStart(2, "0");
  const values = [pis, cofins].filter((value) => value !== "00");
  const same = values.length <= 1 || values.every((value) => value === values[0]);

  if (!values.length) {
    return {
      status: "attention",
      title: "PIS/COFINS sem CST",
      detail: "Informe CST de PIS e COFINS para validar a coerencia com o regime tributario.",
    };
  }

  if (!same) {
    return {
      status: "warning",
      title: "PIS e COFINS divergentes",
      detail: `PIS CST ${pis} e COFINS CST ${cofins} diferem; confirme se ha regra especifica para a operacao.`,
    };
  }

  const code = values[0];

  if (normalizedRegime.includes("simples")) {
    if (code === "99") {
      return {
        status: "ok",
        title: "Simples Nacional coerente",
        detail: "Para Simples Nacional, CST 99 costuma ser usado como classificacao operacional de outras saidas.",
      };
    }
    return {
      status: "attention",
      title: "Revisar PIS/COFINS no Simples",
      detail: `CST ${code} pode exigir parametrizacao especifica, pois empresas do Simples geralmente usam CST 99 para PIS/COFINS.`,
    };
  }

  if (normalizedRegime.includes("presumido") || normalizedRegime.includes("cumulativo")) {
    if (["01", "02", "07", "08", "09", "49"].includes(code)) {
      return {
        status: "ok",
        title: "Regime cumulativo coerente",
        detail: `CST ${code} e compativel com operacoes comuns do regime cumulativo, sujeito a natureza do produto e operacao.`,
      };
    }
    return {
      status: "warning",
      title: "CST incomum para cumulativo",
      detail: `CST ${code} normalmente pede revisao no Lucro Presumido/cumulativo, especialmente se for codigo de credito de entrada.`,
    };
  }

  if (normalizedRegime.includes("real") || normalizedRegime.includes("nao cumulativo") || normalizedRegime.includes("nao-cumulativo")) {
    if (["01", "02", "03", "04", "05", "06", "07", "08", "09", "49", "50", "51", "52", "53", "54", "55", "56"].includes(code)) {
      return {
        status: "ok",
        title: "Regime nao cumulativo coerente",
        detail: `CST ${code} pode ser usado no regime nao cumulativo, separando saida tributada e entradas com credito quando aplicavel.`,
      };
    }
    return {
      status: "attention",
      title: "Revisar CST no nao cumulativo",
      detail: `CST ${code} exige validacao da natureza da operacao, credito e receita vinculada.`,
    };
  }

  return {
    status: "attention",
    title: "Regime nao informado",
    detail: `CST ${code} foi identificado, mas o regime tributario precisa ser informado para validar PIS/COFINS.`,
  };
}

export function getICMSRegimeAdvice(regime: unknown, crt: unknown, icmsCST: unknown, csosn: unknown): ICMSRegimeAdvice {
  const normalizedRegime = String(regime || "")
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase();
  const normalizedCRT = String(crt || "").replace(/\D/g, "");
  const cstDigits = String(icmsCST || "").replace(/\D/g, "");
  const csosnDigits = String(csosn || "").replace(/\D/g, "");
  const isSimple = normalizedCRT === "1" || normalizedRegime.includes("simples");

  if (isSimple) {
    if (csosnDigits) {
      const info = getCSOSNInfo(csosnDigits);
      return {
        status: "ok",
        title: "ICMS do Simples por CSOSN",
        detail: `CRT 1/Simples Nacional deve usar CSOSN. Codigo ${csosnDigits}${info ? ` - ${info.label}` : ""}.`,
      };
    }
    if (cstDigits) {
      return {
        status: "warning",
        title: "Simples usando CST ICMS",
        detail: "Empresas do Simples Nacional devem usar CSOSN, nao CST ICMS de regime normal.",
      };
    }
    return {
      status: "attention",
      title: "CSOSN pendente",
      detail: "Informe CSOSN para validar ICMS de empresa no Simples Nacional.",
    };
  }

  if (cstDigits) {
    const info = getICMSCSTInfo(cstDigits);
    const origin = getICMSOriginInfo(cstDigits);
    return {
      status: cstDigits.length >= 3 ? "ok" : "attention",
      title: cstDigits.length >= 3 ? "CST ICMS completo" : "CST ICMS sem origem",
      detail: `${cstDigits.length >= 3 ? `Origem: ${origin || "nao identificada"}. ` : "O CST de ICMS do regime normal idealmente tem 3 digitos: origem + tributacao. "}${info ? `Tributacao: ${info.label}.` : "Tributacao nao identificada."}`,
    };
  }

  if (csosnDigits) {
    return {
      status: "warning",
      title: "Regime normal usando CSOSN",
      detail: "Lucro Real/Presumido normalmente usa CST ICMS de 3 digitos, nao CSOSN.",
    };
  }

  return {
    status: "attention",
    title: "ICMS sem CST/CSOSN",
    detail: "Informe CST ICMS para regime normal ou CSOSN para Simples Nacional.",
  };
}

export const icmsCSTReference = Object.values(ICMS_CST);
export const csosnReference = Object.values(CSOSN);
export const pisCofinsCSTReference = Object.values(PIS_COFINS_CST);
export const ipiCSTReference = Object.values(IPI_CST);
