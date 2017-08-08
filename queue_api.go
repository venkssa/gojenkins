package gojenkins

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

type QueueStats struct {
	Length    uint32
	TaskNames []string
}

type QueueID uint32

type QueueItem struct {
	Number BuildNumber
	URL    string
}

const (
	// DefaultWaitForBuildToBeQueuedTimeout is the default time that the client would poll before giving up.
	DefaultWaitForBuildToBeQueuedTimeout = time.Duration(1 * time.Minute)
)

// QueueAPI is the interface to interact with Jenkins Queue api.
type QueueAPI interface {
	QueueStats(ctx context.Context) (QueueStats, error)

	// WaitUntilBuildIsQueued polls jenkins queue api until the build starts to execute.
	// A timeout is enforced via context.
	// If the context does not have a timeout DefaultWaitForBuildToBeQueuedTimeout is used.
	// To not bombard jenkins after every unsuccessful call we wait for retryAfter before retrying.
	WaitUntilBuildIsQueued(ctx context.Context, id QueueID, retryAfter time.Duration) (QueueItem, error)
}

func NewQueueAPI(u URLBuilder, r Requestor) queueAPI {
	return queueAPI{u, r}
}

type queueAPI struct {
	URLBuilder
	requestor Requestor
}

func (q queueAPI) QueueStats(ctx context.Context) (QueueStats, error) {
	var queueResponse struct {
		Items []struct {
			Task struct {
				Name string
			}
		}
	}

	resp := q.requestor.Do(ctx, Request{
		Method: http.MethodGet,
		URL:    q.URLBuilder.JSONEndpoint("/queue"),
	})

	if err := resp.VerifyAndDecode(JsonDecoder(&queueResponse)); err != nil {
		return QueueStats{}, err
	}

	stats := QueueStats{Length: uint32(len(queueResponse.Items))}
	for _, item := range queueResponse.Items {
		stats.TaskNames = append(stats.TaskNames, item.Task.Name)
	}

	return stats, nil
}

func (q queueAPI) WaitUntilBuildIsQueued(ctx context.Context, id QueueID, retryAfter time.Duration) (QueueItem, error) {
	ctx, cancelFn := setTimeoutIfNotSet(ctx, DefaultWaitForBuildToBeQueuedTimeout)
	defer cancelFn()

	var queueItem struct {
		Executable struct {
			Number BuildNumber
			URL    string
		}
	}
	url := q.URLBuilder.JSONEndpoint("queue", "item", strconv.FormatUint(uint64(id), 10))

	err := retryUntilFalseOrError(ctx, retryAfter, func() (bool, error) {
		resp := q.requestor.Do(ctx, Request{Method: http.MethodGet, URL: url})
		err := resp.VerifyAndDecode(JsonDecoder(&queueItem))
		return queueItem.Executable.Number == 0, err
	})

	return QueueItem(queueItem.Executable), err
}
