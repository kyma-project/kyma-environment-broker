package httputil

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseRecorder_Write(t *testing.T) {
	t.Run("write data and update size", func(t *testing.T) {
		// given
		recorder := httptest.NewRecorder()
		responseRecorder := NewResponseRecorder(recorder)
		data := []byte("testing response writer")

		// when
		size, err := responseRecorder.Write(data)

		// then
		assert.NoError(t, err)
		assert.Equal(t, len(data), size)
		assert.Equal(t, len(data), responseRecorder.Size)
		assert.Equal(t, data, recorder.Body.Bytes())
	})
}
