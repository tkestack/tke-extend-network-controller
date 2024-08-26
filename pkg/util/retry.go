package util

import (
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func RequeueIfConflict(err error) (ctrl.Result, error) {
	if apierrors.IsConflict(err) {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, err
}

func RetryIfTooManyRequests(fn func() error) error {
	for {
		err := fn()
		if err == nil {
			return nil
		}
		if apierrors.IsTooManyRequests(err) {
			time.Sleep(time.Second)
			continue
		}
		return err
	}
}

func RetryIfPossible(fn func() error) error {
	for {
		err := fn()
		if err == nil {
			return nil
		}
		if apierrors.IsTooManyRequests(err) || apierrors.IsConflict(err) { // API限速或资源冲突时，尝试重试
			time.Sleep(time.Second)
			continue
		}
		return err
	}
}
