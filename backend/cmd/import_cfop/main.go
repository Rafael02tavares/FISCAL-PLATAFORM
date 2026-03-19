package main

import (
	"context"
	"encoding/csv"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	file := flag.String("file", "data/cfop/cfop.csv", "arquivo cfop")
	db := flag.String("database", "postgres://postgres:postgres@127.0.0.1:5432/fiscal_platform?sslmode=disable", "db")
	flag.Parse()

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, *db)
	if err != nil {
		log.Fatal(err)
	}

	defer pool.Close()

	f, err := os.Open(*file)
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(f)
	reader.Comma = ';'

	records, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}

	defer tx.Rollback(ctx)

	for i, r := range records {

		if i == 0 {
			continue
		}

		code := normalizeCFOP(r[0])
		description := r[1]

		indNFe := r[2] == "1"
		indComunica := r[3] == "1"
		indTransp := r[4] == "1"
		indDevol := r[5] == "1"

		_, err := tx.Exec(ctx,
			`INSERT INTO cfop_catalog 
			(code, description, ind_nfe, ind_comunication, ind_transport, ind_devolution)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (code) DO UPDATE SET description=EXCLUDED.description`,
			code,
			description,
			indNFe,
			indComunica,
			indTransp,
			indDevol,
		)

		if err != nil {
			log.Fatal(err)
		}

	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("CFOP importado com sucesso")
}

func normalizeCFOP(v string) string {

	v = strings.ReplaceAll(v, " ", "")
	v = strings.TrimSpace(v)

	return v
}