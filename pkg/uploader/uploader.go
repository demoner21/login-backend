package uploader

import (
	"fmt"
	"image"
	"image/jpeg"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"loginbackend/config"
)

// UploadProfilePicture recebe o ID do usuário para criar um nome de arquivo fixo (Canonical Name)
func UploadProfilePicture(userID string, file multipart.File, header *multipart.FileHeader, cfg *config.Config) (string, error) {
	// 1. Decodificar a imagem (detecta se é PNG ou JPEG)
	img, format, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("erro ao decodificar imagem: %v", err)
	}

	// Validação básica de formato
	if format != "jpeg" && format != "png" {
		return "", fmt.Errorf("formato não suportado: %s (use jpg ou png)", format)
	}

	// 2. Definir nome CANÔNICO (Sempre userID.jpg)
	filename := fmt.Sprintf("%s.jpg", userID)

	if cfg.UploadProvider == "local" {
		return saveLocalFixed(img, filename, cfg)
	}

	return "", fmt.Errorf("provider desconhecido")
}

func saveLocalFixed(img image.Image, filename string, cfg *config.Config) (string, error) {
	// Garantir diretório
	if _, err := os.Stat(cfg.UploadDir); os.IsNotExist(err) {
		os.MkdirAll(cfg.UploadDir, os.ModePerm)
	}

	filePath := filepath.Join(cfg.UploadDir, filename)

	// 3. Criar/Sobrescrever o arquivo
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// 4. Re-encodar tudo para JPEG (Padronização e Compressão)
	// Quality 85 é um excelente balanço entre tamanho e qualidade
	opts := &jpeg.Options{Quality: 85}
	if err := jpeg.Encode(dst, img, opts); err != nil {
		return "", fmt.Errorf("erro ao converter para jpg: %v", err)
	}

	// Retorna URL.
	// Nota: Adicionamos um timestamp fictício aqui se quisermos forçar o refresh no retorno imediato,
	// mas o ideal é o frontend gerenciar o cache busting.
	baseURL := strings.TrimRight(cfg.StorageURL, "/")
	fullURL := fmt.Sprintf("%s/%s", baseURL, filename)
	return fullURL, nil
}
