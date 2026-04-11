package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Claims embedded in the JWT.
type Claims struct {
	UserID   uuid.UUID `json:"uid"`
	Username string    `json:"usr"`
	jwt.RegisteredClaims
}

// User is returned after registration.
type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}

// Service handles registration, login, and token validation.
type Service struct {
	secret []byte
	pool   *pgxpool.Pool
}

func NewService(secret string, pool *pgxpool.Pool) *Service {
	return &Service{secret: []byte(secret), pool: pool}
}

// Register creates a new user with profile, loadout, and starter items.
func (s *Service) Register(ctx context.Context, username, email, password string) (*User, error) {
	if len(username) < 3 || len(username) > 20 {
		return nil, fmt.Errorf("username must be 3-20 characters")
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id`,
		username, email, string(hash),
	).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("username or email already taken")
	}

	if _, err = tx.Exec(ctx, `INSERT INTO profiles (user_id) VALUES ($1)`, userID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO loadouts (user_id) VALUES ($1)`, userID); err != nil {
		return nil, err
	}

	// Starter inventory.
	starterItems := []string{"wpn_laser_1", "shld_basic", "hull_basic", "ai_core_1"}
	for _, itemID := range starterItems {
		if _, err = tx.Exec(ctx, `INSERT INTO inventory (user_id, item_id) VALUES ($1, $2)`, userID, itemID); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &User{ID: userID, Username: username}, nil
}

// Login validates credentials and returns a signed JWT.
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	var userID uuid.UUID
	var username, passwordHash string
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash FROM users WHERE email = $1`, email,
	).Scan(&userID, &username, &passwordHash)
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken parses and validates a JWT, returning the claims.
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
