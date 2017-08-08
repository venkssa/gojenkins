package gojenkins

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestJobApi_ScheduleBuild(t *testing.T) {
	api, cleanupFn := jobAPITestClient("/job/Test/buildWithParameters/api/json",
		func(resp http.ResponseWriter, req *http.Request) {
			resp.Header().Add("Location", "http://testurl.com/queue/item/3")
			resp.WriteHeader(http.StatusCreated)
		})
	defer cleanupFn()

	queueID, err := api.ScheduleBuild(context.TODO(), "Test", url.Values{})
	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}
	if queueID != 3 {
		t.Errorf("Expected 2 but got %v", queueID)
	}
}

func TestJobApi_GetBuilds(t *testing.T) {
	var actualRequest *http.Request
	api, cleanupFn := jobAPITestClient("/job/Test/api/json", func(resp http.ResponseWriter, req *http.Request) {
		actualRequest = req
		fmt.Fprint(resp, getBuildsResponse)
	})
	defer cleanupFn()

	buildInfos, err := api.GetBuilds(context.TODO(), "Test", 0, 5)
	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}
	expectedRawQuery := url.Values{"tree": []string{fmt.Sprintf("builds[%v]{0,5}", buildInfoTree)}}.Encode()
	if actualRequest.URL.RawQuery != expectedRawQuery {
		t.Errorf("Expected %v but got %v", expectedRawQuery, actualRequest.URL.RawQuery)
	}

	if len(buildInfos) != 3 {
		t.Errorf("Expected 3 builds but got %v", len(buildInfos))
	}
}

func TestJobApi_WaitUntilBuildIsComplete(t *testing.T) {
	tests := map[string]struct {
		jenkinsResponses  []string
		expectedBuildInfo BuildInfo
	}{
		"should stop onces jenkins response as building is complete": {
			jenkinsResponses: []string{buildCompleteResponse},
			expectedBuildInfo: BuildInfo{
				Number: 2, QueueID: 3, Result: "SUCCESS", URL: "http://testurl.com/jenkins/job/Test/2"},
		},
		"should retry until jenkins build completes": {
			jenkinsResponses: []string{buildInProgressResponse, buildCompleteResponse},
			expectedBuildInfo: BuildInfo{
				Number: 2, QueueID: 3, Result: "SUCCESS", URL: "http://testurl.com/jenkins/job/Test/2"},
		},
	}

	for testName, testdata := range tests {
		t.Run(testName, func(t *testing.T) {
			info, err := launchAndWaitUntilBuildIsComplete(
				responseCountCheckingHandlerFunc(t, testdata.jenkinsResponses...),
				50*time.Millisecond,
				1*time.Millisecond)
			if err != nil {
				t.Fatalf("Expected no error but got %v", err)
			}
			if info != testdata.expectedBuildInfo {
				t.Errorf("Expected %v but got %v", testdata.expectedBuildInfo, info)
			}
		})
	}
}

func TestJobApi_WaitUntilBuildIsComplete_StopsPollingOnError(t *testing.T) {
	_, err := launchAndWaitUntilBuildIsComplete(responseCountCheckingHandlerFunc(t, ""), 50*time.Millisecond, 1*time.Millisecond)

	if !strings.Contains(err.Error(), "EOF") {
		t.Fatalf("Expected context deadline exceeded error but received %v", err)
	}
}

func TestJobApi_WaitUntilBuildIsComplete_TimesoutCorrectly(t *testing.T) {
	_, err := launchAndWaitUntilBuildIsComplete(stringResponseHandleFunc(buildInProgressResponse), 1*time.Millisecond, 2*time.Millisecond)

	if !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Fatalf("Expected '... %v' error but received '%v'", context.DeadlineExceeded, err)
	}
}

func launchAndWaitUntilBuildIsComplete(fn http.HandlerFunc, timeout time.Duration, retryAfter time.Duration) (BuildInfo, error) {
	client, cleanupFn := jobAPITestClient("/job/Test/1/api/json", fn)
	defer cleanupFn()

	item := QueueItem{URL: fmt.Sprintf("%v/job/Test/1", client.URLBuilder)}

	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	return client.WaitUntilBuildIsComplete(ctx, item, retryAfter)
}

func jobAPITestClient(path string, fn http.HandlerFunc) (jobAPI, func()) {
	mux := http.NewServeMux()
	mux.HandleFunc(path, fn)

	srvr := httptest.NewServer(mux)
	api := NewJobAPI(URLBuilder(srvr.URL), BasicAuthRequestor("", ""))
	return api, srvr.Close
}

const getBuildsResponse = `
{
    "builds": [{
        "number": 2,
        "queueId": 3,
        "result": "SUCCESS",
        "url": "http://testurl.com/jenkins/job/Test/2"
    }, {
        "number": 3,
        "queueId": 4,
        "result": "FAILURE",
        "url": "http://testurl.com/jenkins/job/Test/3"
    }, {
        "number": 4,
        "queueId": 5,
        "result": "SUCCESS",
        "url": "http://testurl.com/jenkins/job/Test/4"
    }]
}
`

const buildInProgressResponse = `
{
  "building" : true,
  "number" : 2,
  "queueId" : 3,
  "result" : "SUCCESS",
  "url" : "http://testurl.com/jenkins/job/Test/2"
}
`

const buildCompleteResponse = `
{
  "building" : false,
  "number" : 2,
  "queueId" : 3,
  "result" : "SUCCESS",
  "url" : "http://testurl.com/jenkins/job/Test/2"
}
`
