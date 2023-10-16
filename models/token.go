package models

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"github.com/udovin/gosql"
)

type TokenKind int

const (
	ConfirmEmailToken  TokenKind = 1
	ResetPasswordToken TokenKind = 2
)

type TokenConfig interface {
	TokenKind() TokenKind
}

type ConfirmEmailTokenConfig struct {
	Email string `json:"string"`
}

func (c ConfirmEmailTokenConfig) TokenKind() TokenKind {
	return ConfirmEmailToken
}

// Token represents a token.
type Token struct {
	baseObject
	AccountID  int64     `db:"account_id"`
	Secret     string    `db:"secret"`
	Kind       TokenKind `db:"kind"`
	Config     JSON      `db:"config"`
	CreateTime int64     `db:"create_time"`
	ExpireTime int64     `db:"expire_time"`
}

func (o Token) ScanConfig(config TokenConfig) error {
	return json.Unmarshal(o.Config, config)
}

// SetConfig updates kind and config of token.
func (o *Token) SetConfig(config TokenConfig) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Kind = config.TokenKind()
	o.Config = raw
	return nil
}

// Clone creates copy of scope.
func (o Token) Clone() Token {
	return o
}

// GenerateSecret generates a new value for token secret.
func (o *Token) GenerateSecret() error {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return err
	}
	o.Secret = hex.EncodeToString(bytes)
	return nil
}

// TokenEvent represents a token event.
type TokenEvent struct {
	baseEvent
	Token
}

// Object returns event temporary token.
func (e TokenEvent) Object() Token {
	return e.Token
}

// SetObject sets event temporary token.
func (e *TokenEvent) SetObject(o Token) {
	e.Token = o
}

// TokenStore represents store for tokens.
type TokenStore struct {
	baseStore[Token, TokenEvent, *Token, *TokenEvent]
}

// NewTokenStore creates a new instance of TokenStore.
func NewTokenStore(
	db *gosql.DB, table, eventTable string,
) *TokenStore {
	impl := &TokenStore{}
	impl.baseStore = makeBaseStore[Token, TokenEvent](db, table, eventTable)
	return impl
}
