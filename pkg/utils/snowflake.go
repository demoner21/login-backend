package utils

import (
	"strconv"
	"sync"

	"github.com/bwmarrin/snowflake"
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
