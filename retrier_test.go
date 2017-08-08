package gojenkins

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryUntilFalseOrError(t *testing.T) {
	tests := map[string]struct {
		fns           []func() (bool, error)
		timeout       time.Duration
		retryAfter    time.Duration
		expectedError error
	}{
		"First done call should not retry": {
			fns:     wrapRetryFns(doneFunc),
			timeout: 1 * time.Hour,
		},
		"First done with error call should not retry": {
			fns:           wrapRetryFns(doneWithErrorFunc),
			timeout:       1 * time.Hour,
			expectedError: errDone,
		},
		"Successful retry should return immediately": {
			fns:     wrapRetryFns(retryFunc, doneFunc),
			timeout: 1 * time.Hour,
		},
		"Error on retry should return immediately": {
			fns:           wrapRetryFns(retryFunc, doneWithErrorFunc),
			timeout:       1 * time.Hour,
			expectedError: errDone,
		},
		"Deadline exceeded should not retry": {
			fns:           wrapRetryFns(retryFunc),
			timeout:       1 * time.Nanosecond,
			retryAfter:    1 * time.Hour,
			expectedError: context.DeadlineExceeded,
		},
	}

	for testName, testdata := range tests {
		t.Run(testName, func(t *testing.T) {
			ctx, cancelFn := context.WithTimeout(context.Background(), testdata.timeout)
			defer cancelFn()

			mf := newMultiFunc(t, testdata.fns...)
			err := retryUntilFalseOrError(ctx, testdata.retryAfter, mf.Fn)

			if err != testdata.expectedError {
				t.Errorf("Expected %v but got %v", testdata.expectedError, err)
			}
			mf.ValidateCalls()
		})
	}

}

func wrapRetryFns(fns ...func() (bool, error)) []func() (bool, error) {
	return fns
}

func doneFunc() (bool, error) {
	return false, nil
}

func retryFunc() (bool, error) {
	return true, nil
}

var errDone = errors.New("don't retry error func")

func doneWithErrorFunc() (bool, error) {
	return false, errDone
}

var errRetry = errors.New("retry true with error")

func retryWithErrorFunc() (bool, error) {
	return true, errRetry
}

type multiFunc struct {
	t     *testing.T
	fns   []func() (bool, error)
	count int
}

func newMultiFunc(t *testing.T, fns ...func() (bool, error)) *multiFunc {
	return &multiFunc{t, fns, 0}
}

func (mf *multiFunc) Fn() (bool, error) {
	if mf.count >= len(mf.fns) {
		mf.t.Fatalf("Unexpected call, expected %d calls but got %d call", len(mf.fns), mf.count)
	}
	v, err := mf.fns[mf.count]()
	mf.count++
	return v, err
}

func (mf *multiFunc) ValidateCalls() {
	if mf.count != len(mf.fns) {
		mf.t.Errorf("Expected call count 2 but was %v", mf.count)
	}
}
