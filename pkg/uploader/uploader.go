package uploader

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"loginbackend/config"
)

// UploadFile gerencia o salvamento do arquivo (abstrato para local/cloud)
func UploadFile(file multipart.File, header *multipart.FileHeader, cfg *config.Config) (string, error) {
	// 1. Validar extensão (segurança básica)
	ext := filepath.Ext(header.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return "", fmt.Errorf("formato não permitido (use jpg, jpeg ou png)")
	}

	// 2. Gerar nome único (User ID + Timestamp ou Random)
	// Ex: avatar-123456789.jpg
	filename := fmt.Sprintf("avatar-%d%s", time.Now().UnixNano(), ext)

	// LÓGICA DE PROVEDOR (Switch case simples para agora)
	if cfg.UploadProvider == "local" {
		return saveLocal(file, filename, cfg)
	}

	// Futuro: else if cfg.UploadProvider == "s3" { ... }

	return "", fmt.Errorf("provider de upload desconhecido")
}

func saveLocal(file multipart.File, filename string, cfg *config.Config) (string, error) {
	// Criar diretório se não existir
	if _, err := os.Stat(cfg.UploadDir); os.IsNotExist(err) {
		os.MkdirAll(cfg.UploadDir, os.ModePerm)
	}

	// Caminho completo
	filePath := filepath.Join(cfg.UploadDir, filename)

	// Criar arquivo vazio no destino
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// Copiar bytes do upload para o destino
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	// Retornar URL completa para o frontend
	// Ex: http://localhost:8080/uploads/avatar-123.jpg
	fullURL := fmt.Sprintf("%s/uploads/%s", cfg.AppURL, filename)
	return fullURL, nil
}
