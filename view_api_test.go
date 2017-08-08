package gojenkins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestViewApi_ListJobNames(t *testing.T) {
	api, cleanupFn := viewAPITestClient("/view/test-view/api/json", stringResponseHandleFunc(listJobsResponse))
	defer cleanupFn()

	names, err := api.ListJobNames(context.TODO(), "test-view")

	if err != nil {
		t.Fatalf("Expected a list of jobs but got error %v", err)
	}

	expectedNames := []string{"test-job-1", "test-job-2", "test-job-3"}

	if !reflect.DeepEqual(expectedNames, names) {
		t.Errorf("Expected %v but got %v", expectedNames, names)
	}
}

func viewAPITestClient(path string, fn http.HandlerFunc) (viewAPI, func()) {
	mux := http.NewServeMux()
	mux.HandleFunc(path, fn)

	srvr := httptest.NewServer(mux)
	api := NewViewAPI(URLBuilder(srvr.URL), BasicAuthRequestor("", ""))
	return api, func() {
		srvr.Close()
	}
}

const listJobsResponse = `
{
  "description" : "Test View.",
  "jobs" : [
    {
      "name" : "test-job-1",
      "url" : "http://testurl.com/jenkins/job/test-job-1",
      "color" : "blue"
    },
    {
      "name" : "test-job-2",
      "url" : "http://testurl.com/jenkins/job/test-job-2",
      "color" : "blue"
    },
    {
      "name" : "test-job-3",
      "url" : "http://testurl.com/jenkins/job/test-job-3",
      "color" : "blue"
    } 
  ],
  "name" : "test-view",
  "property" : [
    
  ],
  "url" : "http://testurl.com/jenkins/view/test-view/"
 }
`
