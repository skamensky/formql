package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/skamensky/formql/pkg/formql/schema"
)

func TestStaticProviderOverridesBaseTable(t *testing.T) {
	provider := StaticProvider{
		Snapshot: &Snapshot{
			Catalog: testCatalog(),
		},
	}

	snapshot, err := provider.Load(context.Background(), Ref{BaseTable: "customers"})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if snapshot.Catalog.BaseTable != "customers" {
		t.Fatalf("expected overridden base table, got %q", snapshot.Catalog.BaseTable)
	}
}

func TestCachingProviderCachesUntilTTLExpires(t *testing.T) {
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	upstreamCalls := 0
	upstream := providerFunc(func(_ context.Context, ref Ref) (*Snapshot, error) {
		upstreamCalls++
		return &Snapshot{
			Catalog: &schema.Catalog{
				BaseTable: ref.BaseTable,
				Tables: []schema.Table{
					{Name: ref.BaseTable, Columns: []schema.Column{{Name: "id", Type: schema.TypeNumber}}},
				},
			},
		}, nil
	})

	cache := &MemoryCache{}
	provider := CachingProvider{
		Upstream: upstream,
		Cache:    cache,
		TTL:      time.Minute,
		Now: func() time.Time {
			return now
		},
	}

	ref := Ref{BaseTable: "orders"}
	if _, err := provider.Load(context.Background(), ref); err != nil {
		t.Fatalf("first Load returned error: %v", err)
	}
	if _, err := provider.Load(context.Background(), ref); err != nil {
		t.Fatalf("second Load returned error: %v", err)
	}
	if upstreamCalls != 1 {
		t.Fatalf("expected 1 upstream call, got %d", upstreamCalls)
	}

	now = now.Add(2 * time.Minute)
	if _, err := provider.Load(context.Background(), ref); err != nil {
		t.Fatalf("third Load returned error: %v", err)
	}
	if upstreamCalls != 2 {
		t.Fatalf("expected cache miss after TTL expiry, got %d upstream calls", upstreamCalls)
	}
}

func TestCachingProviderPropagatesErrors(t *testing.T) {
	expected := errors.New("boom")
	provider := CachingProvider{
		Upstream: providerFunc(func(context.Context, Ref) (*Snapshot, error) {
			return nil, expected
		}),
	}

	_, err := provider.Load(context.Background(), Ref{BaseTable: "orders"})
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestMemoryCacheReturnsDeepClone(t *testing.T) {
	cache := &MemoryCache{}
	original := &Snapshot{
		Catalog: testCatalog(),
	}
	cache.Put("orders", original)

	cached, ok := cache.Get("orders")
	if !ok {
		t.Fatal("expected cached snapshot")
	}
	cached.Catalog.Tables[0].Columns[0].Name = "mutated"

	cachedAgain, ok := cache.Get("orders")
	if !ok {
		t.Fatal("expected cached snapshot on second get")
	}
	if got := cachedAgain.Catalog.Tables[0].Columns[0].Name; got != "id" {
		t.Fatalf("expected cached snapshot to be isolated, got %q", got)
	}
}

func TestInspectorBuildsSchemaInfo(t *testing.T) {
	provider := StaticProvider{
		Snapshot: &Snapshot{
			Catalog: testCatalog(),
		},
	}

	info, err := Inspector{Provider: provider}.LoadInfo(context.Background(), Ref{BaseTable: "orders"})
	if err != nil {
		t.Fatalf("LoadInfo returned error: %v", err)
	}
	if info.BaseTable != "orders" {
		t.Fatalf("expected base table orders, got %q", info.BaseTable)
	}
	if len(info.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(info.Tables))
	}
}

func TestDecodeCatalogJSONValidates(t *testing.T) {
	_, err := DecodeCatalogJSON([]byte(`{"base_table":"missing","tables":[],"relationships":[]}`))
	if err == nil {
		t.Fatal("expected validation error")
	}
}

type providerFunc func(ctx context.Context, ref Ref) (*Snapshot, error)

func (f providerFunc) Load(ctx context.Context, ref Ref) (*Snapshot, error) {
	return f(ctx, ref)
}

func testCatalog() *schema.Catalog {
	return &schema.Catalog{
		BaseTable: "orders",
		Tables: []schema.Table{
			{
				Name: "orders",
				Columns: []schema.Column{
					{Name: "id", Type: schema.TypeNumber},
					{Name: "customer_id", Type: schema.TypeNumber},
				},
			},
			{Name: "customers", Columns: []schema.Column{{Name: "id", Type: schema.TypeNumber}}},
		},
		Relationships: []schema.Relationship{
			{
				Name:         "customer",
				FromTable:    "orders",
				ToTable:      "customers",
				JoinColumn:   "customer_id",
				TargetColumn: "id",
			},
		},
	}
}
