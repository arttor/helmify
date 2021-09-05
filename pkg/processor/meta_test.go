package processor

import (
	"github.com/arttor/helmify/internal"
	"github.com/arttor/helmify/pkg/metadata"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProcessObjMeta(t *testing.T) {
	testMeta := metadata.New("chart-name")
	testMeta.Load(internal.TestNs)
	res, err := ProcessObjMeta(testMeta, internal.TestNs)
	assert.NoError(t, err)
	assert.Contains(t, res, "chart-name.labels")
	assert.Contains(t, res, "chart-name.fullname")
}
