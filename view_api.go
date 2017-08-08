package gojenkins

import (
	"context"
	"net/http"
)

type ViewAPI interface {
	ListJobNames(ctx context.Context, viewName string) ([]string, error)
}

func NewViewAPI(u URLBuilder, r Requestor) viewAPI {
	return viewAPI{u, r}
}

type viewAPI struct {
	URLBuilder
	requestor Requestor
}

func (v viewAPI) ListJobNames(ctx context.Context, viewName string) ([]string, error) {
	var jobs struct {
		Jobs []struct {
			Name string
		}
	}

	resp := v.requestor.Do(ctx, Request{
		Method: http.MethodGet,
		URL:    v.URLBuilder.JSONEndpoint("/view", viewName),
	})

	if err := resp.VerifyAndDecode(JsonDecoder(&jobs)); err != nil {
		return []string{}, err
	}

	var names []string
	for _, job := range jobs.Jobs {
		names = append(names, job.Name)
	}
	return names, nil
}
