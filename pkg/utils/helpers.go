package utils

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"golang.org/x/crypto/bcrypt"
)

var (
	node     *snowflake.Node
	nodeOnce sync.Once
)

// InitSnowflake inicializa o gerador de Snowflake ID
func InitSnowflake(nodeID int64) error {
	var err error
	nodeOnce.Do(func() {
		node, err = snowflake.NewNode(nodeID)
	})
	return err
}

// GenerateSnowflakeID gera um novo Snowflake ID
func GenerateSnowflakeID() string {
	if node == nil {
		// Node padrão se não inicializado
		InitSnowflake(1)
	}
	return node.Generate().String()
}

// HashPassword gera um hash bcrypt da senha
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("erro ao gerar hash da senha: %w", err)
	}
	return string(bytes), nil
}

// CheckPasswordHash compara uma senha com seu hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ParseTime converte uma string para time.Time
// Suporta formatos ISO, PostgreSQL e datas simples.
// Sempre retorna o tempo em UTC.
func ParseTime(timeStr string) (time.Time, error) {
	formats := []string{
		// ISO 8601
		time.RFC3339,           // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04:05Z", // 2006-01-02T15:04:05Z
		// PostgreSQL com nanos e offset
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999-07",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		// Apenas data
		"2006-01-02",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, timeStr); err == nil {
			// Garantir que o tempo retornado esteja em UTC
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("não foi possível parsear o tempo: %s", timeStr)
}

// StringToInt64 converte string para int64
func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// Int64ToString converte int64 para string
func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// ParseSnowflakeID converte string para int64
func ParseSnowflakeID(id string) (int64, error) {
	snowflakeID, err := snowflake.ParseString(id)
	if err != nil {
		return 0, err
	}
	return snowflakeID.Int64(), nil
}

// FormatSnowflakeID formata int64 para string
func FormatSnowflakeID(id int64) string {
	return strconv.FormatInt(id, 10)
}
