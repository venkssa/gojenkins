package gojenkins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestQueueApi_QueueStats(t *testing.T) {
	api, cleanupFn := queueAPITestClient("/queue/api/json", stringResponseHandleFunc(queueAPIResponse))
	defer cleanupFn()

	stats, err := api.QueueStats(context.TODO())
	if err != nil {
		t.Fatalf("Expected queue length but got error %v", err)
	}
	expectedStats := QueueStats{1, []string{"test-job-1"}}
	if !reflect.DeepEqual(expectedStats, stats) {
		t.Errorf("Expected stats %v but got %v", expectedStats, stats)
	}
}

func TestQueueAPI_WaitUntilBuildIsQueued(t *testing.T) {
	tests := map[string]struct {
		jenkinsResponses  []string
		expectedQueueItem QueueItem
	}{
		"should query jenkins queue api for the build number": {
			jenkinsResponses:  []string{queueItemWithExecutable},
			expectedQueueItem: QueueItem{2, "http://testurl.com/jenkins/job/test-job-1/2/"},
		},
		"should retry until jenkins queue api returns a response with build nubmer": {
			jenkinsResponses:  []string{"{}", queueItemWithExecutable},
			expectedQueueItem: QueueItem{2, "http://testurl.com/jenkins/job/test-job-1/2/"},
		},
	}

	for testName, testdata := range tests {
		t.Run(testName, func(t *testing.T) {
			queueItem, err := launchAndWaitUntilBuildIsQueued(responseCountCheckingHandlerFunc(t, testdata.jenkinsResponses...), 100*time.Millisecond, 1*time.Millisecond)

			if err != nil {
				t.Errorf("Expected no error but got %v", err)
			}

			if queueItem != testdata.expectedQueueItem {
				t.Errorf("Expected %v but got %v", testdata.expectedQueueItem, queueItem)
			}
		})
	}
}

func TestQueueAPI_WaitUntilBuildIsQueued_StopsPollingOnError(t *testing.T) {
	_, err := launchAndWaitUntilBuildIsQueued(responseCountCheckingHandlerFunc(t, ""), 100*time.Millisecond, 1*time.Millisecond)

	if !strings.Contains(err.Error(), "EOF") {
		t.Fatalf("Expected context deadline exceeded error but received %v", err)
	}
}

func TestQueueAPI_WaitUntilBuildIsQueued_TimesoutCorrectly(t *testing.T) {
	_, err := launchAndWaitUntilBuildIsQueued(stringResponseHandleFunc("{}"), 1*time.Millisecond, 2*time.Millisecond)

	if !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Fatalf("Expected '... %v' error but received '%v'", context.DeadlineExceeded, err)
	}
}

func launchAndWaitUntilBuildIsQueued(fn http.HandlerFunc, timeout time.Duration, retryAfter time.Duration) (QueueItem, error) {
	client, cleanupFn := queueAPITestClient("/queue/item/", fn)
	defer cleanupFn()

	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	return client.WaitUntilBuildIsQueued(ctx, 1, retryAfter)
}

func queueAPITestClient(path string, fn http.HandlerFunc) (queueAPI, func()) {
	mux := http.NewServeMux()
	mux.HandleFunc(path, fn)

	srvr := httptest.NewServer(mux)
	api := NewQueueAPI(URLBuilder(srvr.URL), BasicAuthRequestor("", ""))
	return api, srvr.Close
}

const queueAPIResponse = `
{
  "discoverableItems" : [
    
  ],
  "items" : [
    {
      "actions" : [
        {
          "parameters" : [
            {
              "name" : "Branch",
              "value" : "test_branch"
            }
          ]
        },
        {
          "causes" : [
            {
              "shortDescription" : "Started by user test",
              "userId" : "test",
              "userName" : "test"
            }
          ]
        }
      ],
      "blocked" : false,
      "buildable" : true,
      "id" : 2,
      "inQueueSince" : 1488421278987,
      "params" : "\nBranch=test_branch",
      "stuck" : true,
      "task" : {
        "name" : "test-job-1",
        "url" : "http://testurl.com/jenkins/job/test-job-1",
        "color" : "blue"
      },
      "url" : "queue/item/3/",
      "why" : "Waiting for next available executor",
      "buildableStartMilliseconds" : 1488421278987,
      "pending" : false
    }
  ]
}
`

const queueItemWithExecutable = `
{
    "actions": [{
        "parameters": [{
            "name": "Branch",
            "value": "master"
        }]
    }, {
        "causes": [{
            "shortDescription": "Started by user buildslackbot",
            "userId": "buildslackbot",
            "userName": "buildslackbot"
        }]
    }],
    "blocked": false,
    "buildable": false,
    "id": 1,
    "inQueueSince": 1490168864617,
    "params": "\nBranch=master",
    "stuck": false,
    "task": {
        "name": "test-job-1",
        "url": "http://testurl.com/jenkins/job/test-job-1/",
        "color": "blue_anime"
    },
    "url": "queue/item/1/",
    "why": null,
    "cancelled": false,
    "executable": {
        "number": 2,
        "url": "http://testurl.com/jenkins/job/test-job-1/2/",
        "subBuilds": []
    }
}
`
