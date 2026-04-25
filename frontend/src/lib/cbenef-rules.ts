const REQUIRED_CBENEf_UFS = new Set(["SP", "PR", "RS", "SC", "RJ", "GO", "DF", "ES"]);
const BENEFIT_ICMS_CSTS = new Set(["20", "30", "40", "41", "50", "51", "70"]);
const BENEFIT_CSOSN = new Set(["103", "203", "300", "400", "900"]);

export function requiresCBenefByUF(value: unknown): boolean {
  return REQUIRED_CBENEf_UFS.has(String(value || "").trim().toUpperCase());
}

export function hasICMSBenefitSituation(icmsCST: unknown, csosn: unknown): boolean {
  const cst = String(icmsCST || "").replace(/\D/g, "").slice(-2).padStart(2, "0");
  const simple = String(csosn || "").replace(/\D/g, "");
  return BENEFIT_ICMS_CSTS.has(cst) || BENEFIT_CSOSN.has(simple);
}

export function getCBenefAdvice(uf: unknown, icmsCST: unknown, csosn: unknown, cbenef: unknown) {
  const normalizedUF = String(uf || "").trim().toUpperCase();
  const required = requiresCBenefByUF(normalizedUF);
  const benefit = hasICMSBenefitSituation(icmsCST, csosn);
  const hasCode = String(cbenef || "").trim() !== "";

  if (!required) {
    return {
      status: "ok",
      title: "cBenef sem obrigatoriedade estadual mapeada",
      detail: `${normalizedUF || "UF"} nao esta na lista operacional de UFs com exigencia recorrente cadastrada.`,
    };
  }

  if (benefit && !hasCode) {
    return {
      status: "warning",
      title: "cBenef possivelmente obrigatorio",
      detail: `${normalizedUF} exige cBenef em operacoes com beneficio fiscal. CST/CSOSN indica isencao, reducao, diferimento, suspensao ou tratamento diferenciado sem codigo informado.`,
    };
  }

  if (benefit && hasCode) {
    return {
      status: "ok",
      title: "cBenef informado",
      detail: `UF ${normalizedUF} tem exigencia mapeada e o item possui codigo de beneficio informado.`,
    };
  }

  return {
    status: "attention",
    title: "UF monitora cBenef",
    detail: `UF ${normalizedUF} esta na lista de exigencia de cBenef, mas o CST/CSOSN do item nao indica beneficio fiscal evidente.`,
  };
}

export const cbenefRequiredUFs = Array.from(REQUIRED_CBENEf_UFS).sort();
