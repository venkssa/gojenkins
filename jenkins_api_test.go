package gojenkins

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestSetTimeoutIfNotSet_SetAtTimeoutIfContextDoesNotHaveADeadline(t *testing.T) {
	timeout := 5 * time.Hour
	ctx, cancelFn := setTimeoutIfNotSet(context.Background(), timeout)
	defer cancelFn()

	deadline, _ := ctx.Deadline()
	expectedDeadline := time.Now().Add(timeout)
	if expectedDeadline.Before(deadline) {
		t.Fatalf("Expected timeout deadline to be before %v but was %v", expectedDeadline, deadline)
	}
}

func TestSetTimeoutIfNotSet_DoesNotSetAtTimeoutIfContextDoesHasADeadline(t *testing.T) {
	expectedTimeout := 1 * time.Hour
	initialCtx, initialCancelFn := context.WithTimeout(context.Background(), expectedTimeout)
	defer initialCancelFn()

	ctx, cancelFn := setTimeoutIfNotSet(initialCtx, 5*time.Hour)
	defer cancelFn()

	deadline, _ := ctx.Deadline()

	if actualTimeout := deadline.Sub(time.Now()); expectedTimeout-actualTimeout < 0 {
		t.Fatalf("Expected timeout to be %v but was %v", expectedTimeout, actualTimeout)
	}
}

func responseCountCheckingHandlerFunc(t *testing.T, responses ...string) http.HandlerFunc {
	responseIdx := 0
	return func(resp http.ResponseWriter, req *http.Request) {
		if responseIdx >= len(responses) {
			t.Fatalf("Expected %v calls to jenkins api but was called %v times.", len(responses), responseIdx+1)
			return
		}
		fmt.Fprint(resp, responses[responseIdx])
		responseIdx++
	}
}

func stringResponseHandleFunc(responseStr string) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		fmt.Fprint(resp, responseStr)
	}
}
