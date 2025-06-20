package util

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

func RetryIfPossible(fn func() error) error {
	return RetryOnErrors(
		retry.DefaultBackoff,
		fn,
		apierrors.IsConflict,
		apierrors.IsInternalError,
		apierrors.IsTooManyRequests,
		apierrors.IsTimeout,
		apierrors.IsServerTimeout,
		apierrors.IsServiceUnavailable,
	)
}

func RetryOnErrors(backoff wait.Backoff, fn func() error, retriables ...func(error) bool) error {
	var lastErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		err := fn()
		if err == nil {
			return true, nil
		}
		for _, retriable := range retriables {
			if retriable(err) {
				lastErr = err
				return false, nil
			}
		}
		return false, err
	})
	if err == wait.ErrorInterrupted(err) {
		err = lastErr
	}
	return err
}
