package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// need redis save & get token and stored the value
// its used for email verification / lost password token
type Token struct {
	Prefix string
	ctx    context.Context
}

func (t *Token) Create(value string) (string, error) {
	token := uuid.New().String()

	err := Save(t.ctx, t.key(token), value)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (t *Token) Read(token string) (string, error) {
	var value string
	err := Read(t.ctx, t.key(token), &value)
	if err != nil || value == "" {
		return "", fmt.Errorf("cannot read the value")
	}

	return value, nil
}

func (t *Token) key(token string) string {
	return fmt.Sprintf("%s:%s", t.Prefix, token)
}

func NewToken(ctx context.Context, prefix string) *Token {
	return &Token{
		Prefix: prefix,
		ctx:    ctx,
	}
}
