package formql

const DefaultMaxRelationshipDepth = 30

// Options configures compiler and tooling behavior.
type Options struct {
	MaxRelationshipDepth int `json:"max_relationship_depth,omitempty"`
}

// Normalized returns options with defaults applied.
func (o Options) Normalized() Options {
	if o.MaxRelationshipDepth <= 0 {
		o.MaxRelationshipDepth = DefaultMaxRelationshipDepth
	}
	return o
}
