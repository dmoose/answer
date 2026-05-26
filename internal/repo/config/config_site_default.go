//go:build !multisite

package config

import (
	"context"

	"github.com/apache/answer/internal/entity"
)

func (cr configRepo) getConfigFallback(_ context.Context, _ string) (*entity.Config, bool, error) {
	return nil, false, nil
}
