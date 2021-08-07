package processor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_commonPrefix(t *testing.T) {
	assert.Equal(t, commonPrefix("test", "testicle"), "test")
	assert.Equal(t, commonPrefix("testimony", "testicle"), "testi")
	assert.Equal(t, commonPrefix("testimony", "abc"), "")
	assert.Equal(t, commonPrefix("test", ""), "")
	assert.Equal(t, commonPrefix("", ""), "")
	assert.Equal(t, commonPrefix("", "test"), "")
	assert.Equal(t, commonPrefix("багет", "багаж"), "баг")
}
