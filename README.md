-----
About
-----
This is a API client of Jenkins API written in Go.

[![Build Status](https://travis-ci.org/venkssa/gojenkins.svg?branch=master)](https://travis-ci.org/venkssa/gojenkins/)
[![GoDoc](https://godoc.org/github.com/venkssa/gojenkins?status.svg)](https://godoc.org/github.com/venkssa/gojenkins)

-----
Usage
-----

	client := gojenkins.NewClient("http://myjenkins.com", "basicauth_user", "basicauth_apikey")

-------
GoDoc Example
-------
See [`example_jenkins_api_test.go`](https://godoc.org/github.com/venkssa/gojenkins#NewClient)
