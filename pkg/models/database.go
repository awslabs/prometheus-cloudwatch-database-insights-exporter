package models

import (
	"time"
)

type Instance struct {
	ResourceID   string
	Identifier   string
	Engine       Engine
	CreationTime time.Time
	Tags         map[string]string
	Metrics      *Metrics
}

func (instance Instance) GetFilterableFields() map[string]string {
	return map[string]string{
		"identifier": instance.Identifier,
		"engine":     string(instance.Engine),
	}
}

func (instance Instance) GetFilterableTags() map[string]string {
	if instance.Tags == nil {
		return make(map[string]string)
	}
	return instance.Tags
}
