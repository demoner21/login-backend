package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

func RunMigrations(db *sql.DB, migrationPath string) error {
	// Verificar se o diretório de migrations existe
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		log.Printf("⚠️ Diretório de migrations não encontrado: %s", migrationPath)
		return nil
	}

	files, err := filepath.Glob(filepath.Join(migrationPath, "*.sql"))
	if err != nil {
		return fmt.Errorf("erro ao listar migrations: %w", err)
	}

	if len(files) == 0 {
		log.Printf("ℹ️ Nenhum arquivo de migration encontrado em: %s", migrationPath)
		return nil
	}

	sort.Strings(files)

	// Criar tabela de controle de migrations se não existir
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT current_timestamp
		)
	`)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela de migrations: %w", err)
	}

	for _, file := range files {
		version := filepath.Base(file)

		// Verificar se a migration já foi aplicada
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM schema_migrations WHERE version = ?",
			version,
		).Scan(&count)

		if err != nil {
			return fmt.Errorf("erro ao verificar migration: %w", err)
		}

		if count > 0 {
			log.Printf("⏭️ Migration já aplicada: %s", version)
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("erro ao ler migration %s: %w", file, err)
		}

		log.Printf("▶ Executando migration: %s", version)

		// Executar migration em uma transação
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("erro ao iniciar transação: %w", err)
		}

		_, err = tx.Exec(string(content))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("erro na migration %s: %w", version, err)
		}

		// Registrar migration aplicada
		_, err = tx.Exec(
			"INSERT INTO schema_migrations (version) VALUES (?)",
			version,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("erro ao registrar migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("erro ao commitar migration %s: %w", version, err)
		}

		log.Printf("✅ Migration aplicada: %s", version)
	}

	log.Println("🎉 Todas as migrations foram executadas com sucesso!")
	return nil
}
