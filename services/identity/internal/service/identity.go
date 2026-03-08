package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/saitddundar/gordion-vpn/services/identity/internal/storage"
)

type IdentityService struct {
	storage       *storage.Storage
	jwtSecret     []byte
	networkSecret string
	tokenTTL      time.Duration
}

// identity service
func New(storage *storage.Storage, jwtSecret, networkSecret string, tokenDurationHours int) *IdentityService {
	return &IdentityService{
		storage:       storage,
		jwtSecret:     []byte(jwtSecret),
		networkSecret: networkSecret,
		tokenTTL:      time.Duration(tokenDurationHours) * time.Hour,
	}
}

func (s *IdentityService) RegisterNode(ctx context.Context, publicKey, version, peerID, reqSecret string) (nodeID, token string, expiresAt int64, err error) {
	if s.networkSecret != "" && reqSecret != s.networkSecret {
		return "", "", 0, fmt.Errorf("invalid network secret")
	}
	node, err := s.storage.CreateNode(ctx, publicKey, version, peerID)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create node: %w", err)

	}
	tokenString, exp, err := s.generateToken(node.ID.String())
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to generate token: %w", err)
	}

	tokenRecord := &storage.Token{
		Token:     tokenString,
		NodeID:    node.ID,
		ExpiresAt: time.Unix(exp, 0),
	}

	if err := s.storage.CreateToken(ctx, tokenRecord); err != nil {
		return "", "", 0, fmt.Errorf("failed to create token: %w", err)
	}

	return node.ID.String(), tokenString, exp, nil
}

func (s *IdentityService) ValidateToken(ctx context.Context, tokenString string) (nodeID string, valid bool, err error) {

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", false, nil
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", false, nil
	}
	nodeIDStr, ok := claims["node_id"].(string)
	if !ok {
		return "", false, fmt.Errorf("node_id not found in token")
	}
	tokenRecord, err := s.storage.GetTokenByValue(ctx, tokenString)
	if err != nil {
		return "", false, nil
	}
	if time.Now().After(tokenRecord.ExpiresAt) {
		return "", false, nil
	}
	return nodeIDStr, true, nil
}
func (s *IdentityService) GetPublicKey(ctx context.Context, nodeID string) (string, error) {
	node, err := s.storage.GetNodeByID(ctx, nodeID)
	if err != nil {
		return "", fmt.Errorf("node not found: %w", err)
	}
	return node.PublicKey, nil
}

// Generate a JWT token for a node
func (s *IdentityService) generateToken(nodeID string) (string, int64, error) {
	expiresAt := time.Now().Add(s.tokenTTL).Unix()

	// Create claims (custom claims!)
	claims := jwt.MapClaims{
		"node_id": nodeID,
		"exp":     expiresAt,
		"iat":     time.Now().Unix(),
	}
	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign token
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, expiresAt, nil
}
func (s *IdentityService) CleanupExpiredTokens(ctx context.Context) error {
	return s.storage.DeleteExpiredTokens(ctx)
}
