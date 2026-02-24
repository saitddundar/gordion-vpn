package storage

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Storage struct {
	db *sqlx.DB
}

func New(databaseURL string) (*Storage, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

// checks database connectivity for health checks
func (s *Storage) Ping() error {
	return s.db.Ping()
}

func (s *Storage) CreateNode(ctx context.Context, publicKey, version string) (*Node, error) {
	query := `
		INSERT INTO nodes (public_key, version)
		VALUES ($1, $2)
		RETURNING id, public_key, version, created_at, updated_at
	`

	var node Node
	err := s.db.GetContext(ctx, &node, query, publicKey, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	return &node, nil
}

func (s *Storage) GetNodeByID(ctx context.Context, id string) (*Node, error) {
	query := `SELECT * FROM nodes WHERE id = $1`

	var node Node
	err := s.db.GetContext(ctx, &node, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return &node, nil
}

func (s *Storage) GetNodeByPublicKey(ctx context.Context, publicKey string) (*Node, error) {
	query := `SELECT * FROM nodes WHERE public_key = $1`

	var node Node
	err := s.db.GetContext(ctx, &node, query, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return &node, nil
}

func (s *Storage) CreateToken(ctx context.Context, token *Token) error {
	query := `
		INSERT INTO tokens (token, node_id, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := s.db.GetContext(ctx, token, query, token.Token, token.NodeID, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}

	return nil
}

func (s *Storage) GetTokenByValue(ctx context.Context, tokenValue string) (*Token, error) {
	query := `SELECT * FROM tokens WHERE token = $1`

	var token Token
	err := s.db.GetContext(ctx, &token, query, tokenValue)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return &token, nil
}

func (s *Storage) DeleteExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM tokens WHERE expires_at < NOW()`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	return nil
}
