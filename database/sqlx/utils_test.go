package sqlx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscape(t *testing.T) {
	s := "a\x00\n\r\\'\"\x1ab"

	out := escape(s)

	assert.Equal(t, `a\x00\n\r\\\'\"\x1ab`, out)
}
