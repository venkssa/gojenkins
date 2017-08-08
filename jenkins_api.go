// Package gojenkins provides client to interact with Jenkin's REST API.
package gojenkins

import (
	"context"
	"time"
)

// Client is the interface to interactive with Jenkins using its API.
type Client interface {
	JobAPI
	QueueAPI
	ViewAPI
}

func NewClient(baseURL, username, apiKey string) Client {
	urlBuilder := URLBuilder(baseURL)
	requestor := BasicAuthRequestor(username, apiKey)
	return struct {
		JobAPI
		QueueAPI
		ViewAPI
	}{
		JobAPI:   NewJobAPI(urlBuilder, requestor),
		QueueAPI: NewQueueAPI(urlBuilder, requestor),
		ViewAPI:  NewViewAPI(urlBuilder, requestor),
	}
}

func setTimeoutIfNotSet(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); !ok {
		return context.WithTimeout(ctx, timeout)
	}
	return ctx, func() {}
}
