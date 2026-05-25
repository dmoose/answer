//go:build !multisite

package multisite

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/base/constant"
)

func TestSiteIDFromContext_NoopBuild(t *testing.T) {
	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-xyz")
	got := SiteIDFromContext(ctx)
	if got != "" {
		t.Errorf("non-multisite SiteIDFromContext should return empty, got %q", got)
	}
}

func TestSetSiteID_NoopBuild(t *testing.T) {
	type e struct{ SiteID string }
	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-123")
	entity := &e{SiteID: "original"}
	SetSiteID(ctx, entity)
	if entity.SiteID != "original" {
		t.Errorf("non-multisite SetSiteID should be no-op, got %q", entity.SiteID)
	}
}
