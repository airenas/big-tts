package mongo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTables(t *testing.T) {
	assert.Equal(t, []string{"requests", "status", "emailLock"}, Tables())
}
