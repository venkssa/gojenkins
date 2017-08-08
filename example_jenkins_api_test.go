package gojenkins_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/venkssa/gojenkins"
)

func ExampleNewClient() {
	mockJenkins := newMockJenkinsServer("testjob")
	defer mockJenkins.Close()

	client := gojenkins.NewClient(mockJenkins.URL, "username", "apikey")

	// Context with 10 Minute timeout after which the client gives up and returns deadline exceeded error.
	ctx, cancelFn := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancelFn()

	// Schedules a testjob build and returns the queueID on success.
	queueID, err := client.ScheduleBuild(ctx, "testjob", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Waits for the testjob to be queued.
	queueItem, err := client.WaitUntilBuildIsQueued(ctx, queueID, 30*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	// Waits for the testjob build to complete running.
	buildInfo, err := client.WaitUntilBuildIsComplete(ctx, queueItem, 30*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(buildInfo.Number, buildInfo.Result)
	// Output: 1 SUCCESS
}

type mockJenkinsServer struct {
	*httptest.Server
	jobName string
}

func newMockJenkinsServer(jobName string) *mockJenkinsServer {
	var srv = new(mockJenkinsServer)

	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/job/%v/buildWithParameters/api/json", jobName), srv.scheduleBuildHandlerFunc)
	mux.HandleFunc(fmt.Sprintf("/queue/item/1/api/json"), srv.waitUntilBuildIsQueuedHandlerFunc)
	mux.HandleFunc(fmt.Sprintf("/job/%v/1/api/json", jobName), srv.waitUntilBuildIsCompleteHandlerFunc)
	srv.Server = httptest.NewServer(mux)
	srv.jobName = jobName
	return srv
}

func (m *mockJenkinsServer) scheduleBuildHandlerFunc(resp http.ResponseWriter, _ *http.Request) {
	resp.Header().Add("Location", fmt.Sprintf("%v/queue/item/1", m.URL))
	resp.WriteHeader(http.StatusCreated)
}

func (m *mockJenkinsServer) waitUntilBuildIsQueuedHandlerFunc(resp http.ResponseWriter, _ *http.Request) {
	queueItemStr := fmt.Sprintf(`
		{
			"executable": {
				"number": 1,
				"url": "%v/job/%v/1"
			}
		}`,
		m.URL, m.jobName)

	fmt.Fprintf(resp, queueItemStr)
}

func (m *mockJenkinsServer) waitUntilBuildIsCompleteHandlerFunc(resp http.ResponseWriter, _ *http.Request) {
	buildCompleteStr := fmt.Sprintf(`
		{
			"building" : false,
			"number" : 1,
			"queueId" : 1,
			"result" : "SUCCESS",
			"url" : "/job/%v/1"
		}`,
		m.URL)

	fmt.Fprintf(resp, buildCompleteStr)
}
