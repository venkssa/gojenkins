package gojenkins

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const buildInfoTree = "number,queueId,url,result"

type BuildInfo struct {
	Number   BuildNumber
	QueueID  uint32
	URL      string
	Result   string
	Building bool
}

type BuildNumber uint32

const (
	// DefaultWaitForBuildToBeCompletedTimeout is the default time that the client would poll job status before giving up.
	DefaultWaitForBuildToBeCompletedTimeout = time.Duration(25 * time.Minute)
)

// JobAPI is the interface to interact with a Jenkins job.
type JobAPI interface {
	ScheduleBuild(ctx context.Context, jobName string, params url.Values) (QueueID, error)
	GetBuilds(ctx context.Context, jobName string, m, n uint32) ([]BuildInfo, error)
	BuildInfo(ctx context.Context, item QueueItem) (BuildInfo, error)

	// WaitUntilBuildIsComplete polls jenkins job api until the job completes or a timeout occurs.
	// A timeout is enforced via context.
	// If the context does not have a timeout DefaultWaitForBuildToBeCompletedTimeout is used.
	// To not bombard jenkins after every unsuccessful call we wait for retryAfter before retrying.
	WaitUntilBuildIsComplete(ctx context.Context, item QueueItem, retryAfter time.Duration) (BuildInfo, error)
}

func NewJobAPI(u URLBuilder, r Requestor) jobAPI {
	return jobAPI{u, r}
}

type jobAPI struct {
	URLBuilder
	requestor Requestor
}

var queueIDRegex = regexp.MustCompile(`.*/item/(\d+)`)

func (j jobAPI) ScheduleBuild(ctx context.Context, jobName string, params url.Values) (QueueID, error) {
	resp := j.requestor.Do(ctx, Request{
		Method:      http.MethodPost,
		URL:         j.URLBuilder.JSONEndpoint("job", jobName, "buildWithParameters"),
		ContentType: ContentTypeFormURLEncoded,
		Body:        strings.NewReader(params.Encode()),
	})

	var queueID uint64
	queueIDFromLocation := func(resp *http.Response) error {
		submatch := queueIDRegex.FindStringSubmatch(resp.Header.Get("Location"))
		if submatch == nil {
			return errors.New("failed to parse queueID")
		}
		queueID, _ = strconv.ParseUint(submatch[1], 10, 32)
		return nil
	}

	err := resp.VerifyAndDecode(NoOpDecoder, HTTPStatusCodeVerifier(http.StatusCreated), queueIDFromLocation)
	if err != nil {
		return 0, nil
	}

	return QueueID(queueID), nil
}

func (j jobAPI) GetBuilds(ctx context.Context, jobName string, m, n uint32) ([]BuildInfo, error) {
	resp := j.requestor.Do(ctx, Request{
		Method: http.MethodGet,
		URL:    j.URLBuilder.JSONEndpoint("job", jobName),
		Query:  url.Values{"tree": []string{fmt.Sprintf("builds[%v]{%v,%v}", buildInfoTree, m, n)}},
	})

	var buildInfoResponse struct {
		Builds []BuildInfo
	}

	if err := resp.VerifyAndDecode(JsonDecoder(&buildInfoResponse)); err != nil {
		return nil, err
	}

	return buildInfoResponse.Builds, nil
}

func (j jobAPI) WaitUntilBuildIsComplete(ctx context.Context, item QueueItem, retryAfter time.Duration) (BuildInfo, error) {
	ctx, cancelFn := setTimeoutIfNotSet(ctx, DefaultWaitForBuildToBeCompletedTimeout)
	defer cancelFn()

	var buildInfo BuildInfo
	err := retryUntilFalseOrError(ctx, retryAfter, func() (bool, error) {
		var err error
		buildInfo, err = j.BuildInfo(ctx, item)
		return buildInfo.Building, err
	})

	return buildInfo, err
}

func (j jobAPI) BuildInfo(ctx context.Context, item QueueItem) (BuildInfo, error) {
	buildURL := fmt.Sprintf("%v/%v", item.URL, jsonEndpoint)
	query := url.Values{"tree": []string{fmt.Sprintf("%v,building", buildInfoTree)}}
	var buildInfo BuildInfo
	err := j.requestor.
		Do(ctx, Request{Method: http.MethodGet, URL: buildURL, Query: query}).
		VerifyAndDecode(JsonDecoder(&buildInfo))
	return buildInfo, err
}
