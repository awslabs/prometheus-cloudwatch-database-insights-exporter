package models

import (
	"time"
)

type Instance struct {
	ResourceID   string
	Identifier   string
	Engine       Engine
	CreationTime time.Time
	Metrics      *Metrics
}
