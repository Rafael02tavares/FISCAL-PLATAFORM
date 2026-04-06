# Fiscal Platform

Plataforma SaaS voltada para análise fiscal de produtos, com foco em sugestão de enquadramento tributário para operações estaduais e federais.

## Visão geral

O objetivo da Fiscal Platform é auxiliar empresas e times de cadastro fiscal na identificação e sugestão de tributos aplicáveis a produtos, com base em informações como:

- XML da NF-e
- GTIN
- descrição do produto
- NCM
- CEST
- contexto da operação fiscal

A plataforma foi pensada para funcionar como um assistente de apoio à análise tributária, aumentando produtividade e padronização, sem substituir a validação técnica e fiscal humana.

## Principais objetivos

- importar XML de NF-e para enriquecer o cadastro de produtos
- sugerir classificações e enquadramentos fiscais
- apoiar análises de ICMS, ICMS ST, FCP, PIS, COFINS e IPI
- evoluir a base de conhecimento com aprendizado validado
- oferecer uma plataforma web multiusuário para uso corporativo

## Stack do projeto

### Backend
- Go

### Frontend
- Astro

### Banco de dados
- PostgreSQL

### Infraestrutura
- Docker

## Estrutura do repositório

```text
FISCAL-PLATAFORM/
├── backend/
├── frontend/
├── infra/
└── README.md