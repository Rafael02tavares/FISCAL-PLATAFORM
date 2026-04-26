package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rafa/fiscal-platform/backend/internal/importers"
)

func main() {
	_ = godotenv.Load()

	filePath := flag.String("file", "", "caminho do XLSX de ST estadual")
	uf := flag.String("uf", "GO", "UF das regras")
	sourceName := flag.String("source-name", "CONFAZ ST Estadual", "nome da fonte")
	sourceURL := flag.String("source-url", "", "URL oficial da fonte")
	versionLabel := flag.String("version", "", "versao/vigencia da planilha")
	databaseURL := flag.String("database", os.Getenv("DATABASE_URL"), "DATABASE_URL do PostgreSQL")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("informe --file com o caminho do XLSX")
	}
	if *databaseURL == "" {
		log.Fatal("DATABASE_URL nao informado")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("conectar banco: %v", err)
	}
	defer db.Close()

	importer := importers.NewStateICMSSTImporter(db)
	if err := importer.ImportCONFAZXLSX(ctx, *filePath, *sourceName, *versionLabel, *uf, *sourceURL); err != nil {
		log.Fatalf("importar ST estadual: %v", err)
	}

	fmt.Println("importacao de ST estadual concluida")
}
