package gojenkins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Requestor struct {
	username string
	apiKey   string
}

func BasicAuthRequestor(username, apiKey string) Requestor {
	return Requestor{username, apiKey}
}

func (r Requestor) Do(ctx context.Context, rb Request) *Response {
	req, err := rb.BuildHTTPRequest()
	if err != nil {
		return &Response{err: err}
	}
	req.SetBasicAuth(r.username, r.apiKey)

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))

	if err != nil {
		return &Response{err: err}
	}

	return &Response{err: err, response: resp}
}

const (
	ContentTypeJSON           = "application/json"
	ContentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

type Request struct {
	Method      string
	URL         string
	Query       url.Values
	ContentType string
	Body        io.Reader
}

func (r Request) BuildHTTPRequest() (*http.Request, error) {
	req, err := http.NewRequest(r.Method, r.URL, r.Body)
	if err != nil {
		return nil, err
	}
	if len(r.Query) > 0 {
		req.URL.RawQuery = r.Query.Encode()
	}

	if r.ContentType == "" {
		r.ContentType = ContentTypeJSON
	}
	req.Header.Set("Content-Type", r.ContentType)

	return req, nil
}

type Decoder func(io.Reader) error

func JsonDecoder(v interface{}) Decoder {
	return func(r io.Reader) error {
		return json.NewDecoder(r).Decode(v)
	}
}

func NoOpDecoder(r io.Reader) error {
	_, err := io.Copy(ioutil.Discard, r)
	return err
}

type Verifier func(*http.Response) error

var StatusOKVerifier = HTTPStatusCodeVerifier(http.StatusOK)

func HTTPStatusCodeVerifier(statusCode int) Verifier {
	return func(resp *http.Response) error {
		if resp.StatusCode != statusCode {
			return fmt.Errorf("Unexpected status code %v. Expected %v", resp.StatusCode, statusCode)
		}
		return nil
	}
}

type Response struct {
	err                  error
	response             *http.Response
	isResponseBodyClosed bool
}

func (r *Response) VerifyAndDecode(decoder Decoder, verifiers ...Verifier) error {
	err := r.verifyAndDecode(decoder, verifiers...)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (r *Response) verifyAndDecode(decoder Decoder, verifiers ...Verifier) error {
	if r.err != nil {
		return r.err
	}

	if r.isResponseBodyClosed {
		return errors.New("cannot decode from a closed response body")
	}

	if len(verifiers) == 0 {
		verifiers = append(verifiers, StatusOKVerifier)
	}

	var errs errorSlice
	for _, verifier := range verifiers {
		if err := verifier(r.response); err != nil {
			errs = append(errs, err)
		}
	}

	r.isResponseBodyClosed = true
	defer r.response.Body.Close()
	if err := decoder(r.response.Body); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		r.err = errs
		return errs
	}
	return nil
}

type errorSlice []error

func (es errorSlice) Error() string {
	var errs []string
	for _, err := range es {
		errs = append(errs, err.Error())
	}
	return strings.Join(errs, " : ")
}
