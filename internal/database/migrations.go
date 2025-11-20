package database

import (
	"database/sql"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
)

func RunMigrations(db *sql.DB, migrationPath string) error {
	files, err := filepath.Glob(filepath.Join(migrationPath, "*.sql"))
	if err != nil {
		return err
	}

	sort.Strings(files)

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		log.Printf("▶ Executando migration: %s", file)

		if _, err := db.Exec(string(content)); err != nil {
			return err
		}
	}

	log.Println("✅ Migrations concluídas!")
	return nil
}
