package storage

import (
	"time"

	"github.com/google/uuid"
)

type Node struct {
	ID        uuid.UUID `db:"id"`
	PublicKey string    `db:"public_key"`
	Version   string    `db:"version"`
	PeerID    *string   `db:"peer_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Token struct {
	ID        uuid.UUID `db:"id"`
	Token     string    `db:"token"`
	NodeID    uuid.UUID `db:"node_id"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}
