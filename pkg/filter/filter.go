package filter

type Filterable interface {
	GetFilterableFields() map[string]string
	GetFilterableTags() map[string]string
}

type Filter interface {
	ShouldInclude(obj Filterable) bool
	HasFilters() bool
}
