package error_test

import (
	"fmt"
	"testing"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	apierr2 "k8s.io/apimachinery/pkg/api/meta"
)

func TestLastError(t *testing.T) {
	t.Run("report correct reason and component", func(t *testing.T) {
		// given
		dbErr := fmt.Errorf("something: %w", dberr.NotFound("Some NotFound apperror, %s", "Some pkg err"))
		expectDbErr := "something: Some NotFound apperror, Some pkg err"

		timeoutErr := fmt.Errorf("something: %w", fmt.Errorf("operation has reached the time limit: 2h"))
		expectTimeoutMsg := "something: operation has reached the time limit: 2h"

		// when
		dbLastErr := kebError.ReasonForError(dbErr, "s3")
		timeoutLastErr := kebError.ReasonForError(timeoutErr, "s4")

		// then
		assert.Equal(t, dberr.ErrDBNotFound, dbLastErr.GetReason())
		assert.Equal(t, kebError.KebDbDependency, dbLastErr.GetComponent())
		assert.Equal(t, expectDbErr, dbLastErr.Error())
		assert.Equal(t, "s3", dbLastErr.Step)

		assert.Equal(t, kebError.KEBTimeOutCode, timeoutLastErr.GetReason())
		assert.Equal(t, kebError.KEBDependency, timeoutLastErr.GetComponent())
		assert.Equal(t, expectTimeoutMsg, timeoutLastErr.Error())
		assert.Equal(t, "s4", timeoutLastErr.Step)
	})
}

func TestTemporaryErrorToLastError(t *testing.T) {
	t.Run("wrapped temporary error", func(t *testing.T) {
		// given
		err := kebError.LastError{}.
			SetMessage(fmt.Sprintf("Got status %d", 502)).
			SetReason(kebError.HttpStatusCode).
			SetComponent(kebError.ReconcileDependency)
		tempErr := fmt.Errorf("something else: %w", kebError.WrapNewTemporaryError(fmt.Errorf("something: %w", err)))
		expectMsg := fmt.Sprintf("something else: something: Got status %d", 502)

		// when
		lastErr := kebError.ReasonForError(tempErr, "s1")

		// then
		assert.Equal(t, kebError.HttpStatusCode, lastErr.GetReason())
		assert.Equal(t, kebError.ReconcileDependency, lastErr.GetComponent())
		assert.Equal(t, expectMsg, lastErr.Error())
		assert.True(t, kebError.IsTemporaryError(tempErr))
		assert.Equal(t, "s1", lastErr.Step)
	})

	t.Run("new temporary error", func(t *testing.T) {
		// given
		tempErr := fmt.Errorf("something: %w", kebError.NewTemporaryError("temporary error..."))
		expectMsg := "something: temporary error..."

		// when
		lastErr := kebError.ReasonForError(tempErr, "s1")

		// then
		assert.Equal(t, kebError.KEBInternalCode, lastErr.GetReason())
		assert.Equal(t, kebError.KEBDependency, lastErr.GetComponent())
		assert.Equal(t, expectMsg, lastErr.Error())
		assert.True(t, kebError.IsTemporaryError(tempErr))
		assert.Equal(t, "s1", lastErr.Step)
	})
}

func TestNotFoundError(t *testing.T) {
	// given
	err := fmt.Errorf("something: %w", kebError.NewNotFoundError(kebError.ClusterNotFoundCode, kebError.ReconcileDependency))

	// when
	lastErr := kebError.ReasonForError(err, "s2")

	// then
	assert.EqualError(t, lastErr, "something: not found")
	assert.Equal(t, kebError.ClusterNotFoundCode, lastErr.GetReason())
	assert.Equal(t, kebError.ReconcileDependency, lastErr.GetComponent())
	assert.True(t, kebError.IsNotFoundError(err))
	assert.Equal(t, "s2", lastErr.Step)
}

func TestK8SLastError(t *testing.T) {
	// given
	errBadReq := fmt.Errorf("something: %w", apierr.NewBadRequest("bad request here"))
	errUnexpObj := fmt.Errorf("something: %w", &apierr.UnexpectedObjectError{})
	errAmbi := fmt.Errorf("something: %w", &apierr2.AmbiguousResourceError{})
	errNoMatch := fmt.Errorf("something: %w", &apierr2.NoKindMatchError{})

	// when
	lastErrBadReq := kebError.ReasonForError(errBadReq, "s1")
	lastErrUnexpObj := kebError.ReasonForError(errUnexpObj, "s2")
	lastErrAmbi := kebError.ReasonForError(errAmbi, "s3")
	lastErrNoMatch := kebError.ReasonForError(errNoMatch, "s4")

	// then
	assert.EqualError(t, lastErrBadReq, "something: bad request here")
	assert.Equal(t, kebError.Reason("BadRequest"), lastErrBadReq.GetReason())
	assert.Equal(t, kebError.K8sDependency, lastErrBadReq.GetComponent())
	assert.Equal(t, "s1", lastErrBadReq.Step)

	assert.ErrorContains(t, lastErrUnexpObj, "something: unexpected object: ")
	assert.Equal(t, kebError.K8SUnexpectedObjectCode, lastErrUnexpObj.GetReason())
	assert.Equal(t, kebError.K8sDependency, lastErrUnexpObj.GetComponent())
	assert.Equal(t, "s2", lastErrUnexpObj.Step)

	assert.ErrorContains(t, lastErrAmbi, "matches multiple resources or kinds")
	assert.Equal(t, kebError.K8SAmbiguousCode, lastErrAmbi.GetReason())
	assert.Equal(t, kebError.K8sDependency, lastErrAmbi.GetComponent())
	assert.Equal(t, "s3", lastErrAmbi.Step)

	assert.ErrorContains(t, lastErrNoMatch, "something: no matches for kind")
	assert.Equal(t, kebError.K8SNoMatchCode, lastErrNoMatch.GetReason())
	assert.Equal(t, kebError.K8sDependency, lastErrNoMatch.GetComponent())
	assert.Equal(t, "s4", lastErrNoMatch.Step)
}
