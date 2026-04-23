package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/skamensky/formql/pkg/formql/schema"
)

// Ref identifies a catalog request independently of transport or environment.
type Ref struct {
	Namespace string `json:"namespace,omitempty"`
	BaseTable string `json:"base_table"`
}

// Snapshot is a resolved compiler catalog plus optional provider metadata.
type Snapshot struct {
	Catalog   *schema.Catalog `json:"catalog"`
	Revision  string          `json:"revision,omitempty"`
	FetchedAt time.Time       `json:"fetched_at,omitempty"`
}

// Provider resolves a schema snapshot for compiler, editor, or frontend use.
type Provider interface {
	Load(ctx context.Context, ref Ref) (*Snapshot, error)
}

// ManagedProvider is a provider with optional lifecycle cleanup.
type ManagedProvider interface {
	Provider
	Close()
}

// Cache stores resolved snapshots by key.
type Cache interface {
	Get(key string) (*Snapshot, bool)
	Put(key string, snapshot *Snapshot)
	Delete(key string)
	Clear()
}

// Keyer derives cache keys from logical catalog references.
type Keyer interface {
	Key(ref Ref) string
}

// InfoProvider exposes schema information in a frontend-friendly shape.
type InfoProvider interface {
	LoadInfo(ctx context.Context, ref Ref) (*Info, error)
}

// Info is a stable schema info view for editors, frontends, and tooling.
type Info struct {
	Ref           Ref                   `json:"ref"`
	Revision      string                `json:"revision,omitempty"`
	FetchedAt     time.Time             `json:"fetched_at,omitempty"`
	BaseTable     string                `json:"base_table"`
	Tables        []schema.Table        `json:"tables"`
	Relationships []schema.Relationship `json:"relationships"`
}

// StaticProvider serves a single in-memory catalog snapshot.
type StaticProvider struct {
	Snapshot *Snapshot
}

// JSONProvider loads a static catalog snapshot from a JSON document.
type JSONProvider struct {
	Data      []byte
	Revision  string
	FetchedAt time.Time
}

// NoopCloseProvider adapts a plain provider into a managed provider.
type NoopCloseProvider struct {
	Provider Provider
}

// Inspector projects snapshots into frontend-facing schema info.
type Inspector struct {
	Provider Provider
}

// Load returns the configured snapshot, overriding the base table when set.
func (p StaticProvider) Load(_ context.Context, ref Ref) (*Snapshot, error) {
	if p.Snapshot == nil || p.Snapshot.Catalog == nil {
		return nil, fmt.Errorf("static provider is missing a catalog")
	}
	snapshot := cloneSnapshot(*p.Snapshot)
	baseTable := normalize(ref.BaseTable)
	if baseTable != "" {
		snapshot.Catalog.BaseTable = baseTable
	}
	if err := snapshot.Catalog.Validate(); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// Load decodes the configured JSON document into a validated snapshot.
func (p JSONProvider) Load(ctx context.Context, ref Ref) (*Snapshot, error) {
	snapshot, err := SnapshotFromJSON(p.Data)
	if err != nil {
		return nil, err
	}
	snapshot.Revision = p.Revision
	snapshot.FetchedAt = p.FetchedAt
	if snapshot.FetchedAt.IsZero() {
		snapshot.FetchedAt = time.Now().UTC()
	}
	return StaticProvider{Snapshot: snapshot}.Load(ctx, ref)
}

// CachingProvider decorates another provider with optional caching.
type CachingProvider struct {
	Upstream Provider
	Cache    Cache
	Keyer    Keyer
	TTL      time.Duration
	Now      func() time.Time
}

// Load returns a cached snapshot when present and fresh, otherwise it loads
// from the upstream provider and stores the result.
func (p CachingProvider) Load(ctx context.Context, ref Ref) (*Snapshot, error) {
	if p.Upstream == nil {
		return nil, fmt.Errorf("caching provider requires an upstream provider")
	}

	key := DefaultKeyer{}.Key(ref)
	if p.Keyer != nil {
		key = p.Keyer.Key(ref)
	}

	now := time.Now
	if p.Now != nil {
		now = p.Now
	}

	if p.Cache != nil {
		if snapshot, ok := p.Cache.Get(key); ok && snapshot != nil {
			if p.TTL <= 0 || now().Sub(snapshot.FetchedAt) <= p.TTL {
				cloned := cloneSnapshot(*snapshot)
				return &cloned, nil
			}
			p.Cache.Delete(key)
		}
	}

	snapshot, err := p.Upstream.Load(ctx, ref)
	if err != nil {
		return nil, err
	}
	if snapshot == nil || snapshot.Catalog == nil {
		return nil, fmt.Errorf("upstream provider returned an empty snapshot")
	}

	loaded := cloneSnapshot(*snapshot)
	if loaded.FetchedAt.IsZero() {
		loaded.FetchedAt = now()
	}

	if p.Cache != nil {
		stored := cloneSnapshot(loaded)
		p.Cache.Put(key, &stored)
	}

	return &loaded, nil
}

// Invalidate removes a cached snapshot for a specific reference.
func (p CachingProvider) Invalidate(ref Ref) {
	if p.Cache == nil {
		return
	}
	key := DefaultKeyer{}.Key(ref)
	if p.Keyer != nil {
		key = p.Keyer.Key(ref)
	}
	p.Cache.Delete(key)
}

// Clear removes all cached snapshots.
func (p CachingProvider) Clear() {
	if p.Cache == nil {
		return
	}
	p.Cache.Clear()
}

// MemoryCache is a simple in-memory cache implementation.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*Snapshot
}

// Get returns a cached snapshot when present.
func (c *MemoryCache) Get(key string) (*Snapshot, bool) {
	if c == nil || c.entries == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot, ok := c.entries[key]
	if !ok || snapshot == nil {
		return nil, false
	}
	cloned := cloneSnapshot(*snapshot)
	return &cloned, true
}

// Put stores a snapshot.
func (c *MemoryCache) Put(key string, snapshot *Snapshot) {
	if c == nil || snapshot == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]*Snapshot)
	}
	cloned := cloneSnapshot(*snapshot)
	c.entries[key] = &cloned
}

// Delete removes a snapshot by key.
func (c *MemoryCache) Delete(key string) {
	if c == nil || c.entries == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Clear removes every cached entry.
func (c *MemoryCache) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*Snapshot)
}

// DefaultKeyer uses namespace + base table as the stable cache key.
type DefaultKeyer struct{}

// Key derives a stable cache key for a catalog reference.
func (DefaultKeyer) Key(ref Ref) string {
	namespace := normalize(ref.Namespace)
	baseTable := normalize(ref.BaseTable)
	if namespace == "" {
		return baseTable
	}
	return namespace + ":" + baseTable
}

// Close implements ManagedProvider.
func (p NoopCloseProvider) Close() {}

// Load delegates to the wrapped provider.
func (p NoopCloseProvider) Load(ctx context.Context, ref Ref) (*Snapshot, error) {
	if p.Provider == nil {
		return nil, fmt.Errorf("noop close provider is missing a provider")
	}
	return p.Provider.Load(ctx, ref)
}

// LoadInfo projects a provider snapshot into schema info.
func (i Inspector) LoadInfo(ctx context.Context, ref Ref) (*Info, error) {
	if i.Provider == nil {
		return nil, fmt.Errorf("inspector is missing a provider")
	}
	snapshot, err := i.Provider.Load(ctx, ref)
	if err != nil {
		return nil, err
	}
	return InfoFromSnapshot(snapshot, ref)
}

// DecodeCatalogJSON decodes and validates a compiler catalog from JSON.
func DecodeCatalogJSON(data []byte) (*schema.Catalog, error) {
	var catalog schema.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("decode catalog json: %w", err)
	}
	if err := catalog.Validate(); err != nil {
		return nil, fmt.Errorf("validate catalog: %w", err)
	}
	return &catalog, nil
}

// SnapshotFromJSON creates a static snapshot from catalog JSON data.
func SnapshotFromJSON(data []byte) (*Snapshot, error) {
	catalog, err := DecodeCatalogJSON(data)
	if err != nil {
		return nil, err
	}
	return &Snapshot{
		Catalog: catalog,
	}, nil
}

// InfoFromSnapshot builds frontend-facing schema info from a catalog snapshot.
func InfoFromSnapshot(snapshot *Snapshot, ref Ref) (*Info, error) {
	if snapshot == nil || snapshot.Catalog == nil {
		return nil, fmt.Errorf("snapshot is missing a catalog")
	}
	cloned := cloneSnapshot(*snapshot)
	info := &Info{
		Ref:           normalizeRef(ref, cloned.Catalog.BaseTable),
		Revision:      cloned.Revision,
		FetchedAt:     cloned.FetchedAt,
		BaseTable:     cloned.Catalog.BaseTable,
		Tables:        cloned.Catalog.Tables,
		Relationships: cloned.Catalog.Relationships,
	}
	return info, nil
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	cloned := snapshot
	if snapshot.Catalog != nil {
		catalogCopy := cloneCatalog(*snapshot.Catalog)
		cloned.Catalog = &catalogCopy
	}
	return cloned
}

func cloneCatalog(catalog schema.Catalog) schema.Catalog {
	cloned := catalog
	if catalog.Tables != nil {
		cloned.Tables = make([]schema.Table, len(catalog.Tables))
		for i, table := range catalog.Tables {
			clonedTable := table
			if table.Columns != nil {
				clonedTable.Columns = append([]schema.Column(nil), table.Columns...)
			}
			cloned.Tables[i] = clonedTable
		}
	}
	if catalog.Relationships != nil {
		cloned.Relationships = append([]schema.Relationship(nil), catalog.Relationships...)
	}
	return cloned
}

func normalizeRef(ref Ref, fallbackBaseTable string) Ref {
	return Ref{
		Namespace: normalize(ref.Namespace),
		BaseTable: normalize(coalesce(ref.BaseTable, fallbackBaseTable)),
	}
}

func coalesce(values ...string) string {
	for _, value := range values {
		if normalize(value) != "" {
			return value
		}
	}
	return ""
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
