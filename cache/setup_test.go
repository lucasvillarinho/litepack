package cache

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lucasvillarinho/litepack/database/drivers"
)

func TestCreateIndex(t *testing.T) {

	t.Run("should create index successfully", func(t *testing.T) {
		mock := &drivers.Mock{}
		ch := &cache{
			engine: mock,
		}

		err := createIndex(ch)

		assert.NoError(t, err, "Expected no error when creating index")
		assert.Equal(t, "CREATE INDEX IF NOT EXISTS idx_key ON cache (key);", mock.ExecutedQuery, "Expected query to match")
	})

	t.Run("should return an error when creating index fails", func(t *testing.T) {
		mock := &drivers.Mock{
			QueryErr: fmt.Errorf("mock error"),
		}
		ch := &cache{
			engine: mock,
		}

		err := createIndex(ch)

		assert.Error(t, err, "Expected an error when creating index")
		assert.EqualError(t, err, "creating index: mock error", "Expected error message to match")
	})

}
