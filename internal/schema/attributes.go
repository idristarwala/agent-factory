package schema

// AttributeSet is the set of attributes that apply to a product type.
// A catalog can have hundreds of PTs, each with dozens or hundreds of attributes.
type AttributeSet struct {
	ProductType string   `json:"product_type"`
	Attributes  []string `json:"attributes"`
	Required    []string `json:"required,omitempty"`
}

// AttributeIndex lets the Steward look up which attributes matter for a PT.
type AttributeIndex struct {
	sets map[string]*AttributeSet
}

func NewAttributeIndex() *AttributeIndex {
	return &AttributeIndex{sets: make(map[string]*AttributeSet)}
}

func (a *AttributeIndex) Add(pt string, attrs []string, required []string) {
	a.sets[pt] = &AttributeSet{
		ProductType: pt,
		Attributes:  attrs,
		Required:    required,
	}
}

func (a *AttributeIndex) For(pt string) *AttributeSet {
	return a.sets[pt]
}

func (a *AttributeIndex) All() []*AttributeSet {
	out := make([]*AttributeSet, 0, len(a.sets))
	for _, s := range a.sets {
		out = append(out, s)
	}
	return out
}

// SparseAttributes stores only non-empty values.
type SparseAttributes map[string]any

func (s SparseAttributes) Set(k string, v any) {
	if v == nil || v == "" {
		delete(s, k)
		return
	}
	s[k] = v
}

func (s SparseAttributes) Get(k string) (any, bool) {
	v, ok := s[k]
	return v, ok
}

func (s SparseAttributes) Missing(required []string) []string {
	var miss []string
	for _, r := range required {
		if _, ok := s[r]; !ok {
			miss = append(miss, r)
		}
	}
	return miss
}
