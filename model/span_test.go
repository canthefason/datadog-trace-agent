package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testSpan = Span{
	Duration: 10000000,
	Error:    0,
	Resource: "GET /some/raclette",
	Service:  "django",
	Name:     "django.controller",
	SpanID:   42,
	Start:    1448466874000000000,
	TraceID:  424242,
	Meta: map[string]string{
		"user": "leo",
		"pool": "fondue",
	},
	Metrics: map[string]float64{
		"cheese_weight": 100000.0,
	},
	ParentID: 1111,
	Type:     "http",
}

func TestSpanString(t *testing.T) {
	assert := assert.New(t)
	assert.NotEqual("", testSpan.String())
}

func TestSpanFlushMarker(t *testing.T) {
	assert := assert.New(t)
	s := NewFlushMarker()
	assert.True(s.IsFlushMarker())
}
