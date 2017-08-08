package gojenkins

import "strings"

const (
	jsonEndpoint = "api/json"
)

type URLBuilder string

func (url URLBuilder) JSONEndpoint(paths ...string) string {
	parts := append([]string{string(url)}, append(paths, jsonEndpoint)...)
	return strings.Join(parts, "/")
}
