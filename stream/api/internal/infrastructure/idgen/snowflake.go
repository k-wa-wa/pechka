package idgen

import (
	"encoding/base64"
	"encoding/binary"
	"log"

	"github.com/bwmarrin/snowflake"
	"pechka/streaming-service/api/internal/domain"
)

type snowflakeGenerator struct {
	node *snowflake.Node
}

func NewSnowflakeGenerator(nodeID int64) domain.ShortIDGenerator {
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		log.Fatalf("failed to create snowflake node: %v", err)
	}
	return &snowflakeGenerator{
		node: node,
	}
}

func (g *snowflakeGenerator) Generate() string {
	id := g.node.Generate()
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(id))
	return base64.RawURLEncoding.EncodeToString(b)
}
