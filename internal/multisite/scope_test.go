//go:build multisite

package multisite

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/base/constant"
	"github.com/gin-gonic/gin"
)

func TestSiteIDFromContext_GinContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := &gin.Context{}
	ctx.Set(constant.SiteIDFlag, "site-abc")

	got := SiteIDFromContext(ctx)
	if got != "site-abc" {
		t.Errorf("SiteIDFromContext(gin) = %q, want %q", got, "site-abc")
	}
}

func TestSiteIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := SiteIDFromContext(ctx)
	if got != "" {
		t.Errorf("SiteIDFromContext(empty) = %q, want empty", got)
	}
}

func TestSiteIDFromContext_StdContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-xyz")
	got := SiteIDFromContext(ctx)
	if got != "site-xyz" {
		t.Errorf("SiteIDFromContext(std) = %q, want %q", got, "site-xyz")
	}
}

type testEntity struct {
	SiteID string
	Name   string
}

func TestSetSiteID(t *testing.T) {
	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-123")

	e := &testEntity{Name: "test"}
	SetSiteID(ctx, e)
	if e.SiteID != "site-123" {
		t.Errorf("SetSiteID = %q, want %q", e.SiteID, "site-123")
	}
}

func TestSetSiteID_NoSite(t *testing.T) {
	ctx := context.Background()
	e := &testEntity{Name: "test", SiteID: "original"}
	SetSiteID(ctx, e)
	if e.SiteID != "original" {
		t.Errorf("SetSiteID should not change when no site in ctx, got %q", e.SiteID)
	}
}

func TestSetSiteID_Slice(t *testing.T) {
	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-123")
	items := []*testEntity{{Name: "a"}, {Name: "b"}}
	SetSiteID(ctx, items)
	for _, e := range items {
		if e.SiteID != "site-123" {
			t.Errorf("SetSiteID slice item %s = %q, want %q", e.Name, e.SiteID, "site-123")
		}
	}
}

func TestSetSiteID_NonStruct(t *testing.T) {
	ctx := context.WithValue(context.Background(), constant.SiteIDContextKey, "site-123")
	s := "not a struct"
	SetSiteID(ctx, &s) // should not panic
}
