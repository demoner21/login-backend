package utils

import (
	"fmt"
	"time"
)

// ParseTime converte string para time.Time com formatos comuns do PostgreSQL
func ParseTime(timeStr string) (time.Time, error) {
	// Tentar formatos comuns do PostgreSQL
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05.999999-07",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("não foi possível parsear o tempo: %s", timeStr)
}
