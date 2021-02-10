package main

import (
	"fmt"
	"strconv"
	"strings"
)

const HTTP_POST_METHOD string = "POST"
const HTTP_GET_METHOD string = "GET"
const HTTP_NOT_FOUND string = "Not Found"
const HTTP_NOT_ALLOWED string = "Method Not Allowed"
const HTTP_OK string = "Ok"
const HTTP_NOT_FOUND_BODY string = "404 Not Found"
const HTTP_NOT_ALLOWED_BODY string = "405 Method Not Allowed"
const SPACE string = " "
const NEW_LINE string = "\n"
const HTTP_1_1 string = "HTTP/1.1"
const HEADER_CONTENT_TYPE string = "Content-Type"
const HEADER_CONTENT_LENGTH string = "Content-Length"
const HEADER_CACHE_CONTROL string = "Cache-Control"
const HEADER_CACHE_CONTROL_DEFAULT_VALUE string = "public, max-age=604800"
const HEADER_CONNECTION string = "Connection"
const HEADER_CONNECTION_CLOSE string = "close"
const HEADER_LAST_MODIFIED string = "Last-Modified"
const HEADER_DATE string = "Date"
const HEADER_SERVER string = "Server"
const HEADER_SERVER_VALUE string = "Minosse"

// Response Response structure
type Response struct {
	status     string
	statusCode int
	body       []byte
	protocol   string
	headers    map[string]string
}

func responseMethodNotAllowed() Response {
	return Response{
		status:     HTTP_NOT_FOUND,
		statusCode: 405,
		protocol:   HTTP_1_1,
		body:       []byte(HTTP_NOT_FOUND_BODY),
		headers:    map[string]string{HEADER_CONTENT_TYPE: "text/plain; charset=utf-8"},
	}
}

func responseNotFound() Response {
	return Response{
		status:     HTTP_NOT_FOUND,
		statusCode: 404,
		body:       []byte(HTTP_NOT_ALLOWED_BODY),
		protocol:   HTTP_1_1,
		headers:    map[string]string{HEADER_CONTENT_TYPE: "text/plain; charset=utf-8"},
	}
}

func responseOk(body []byte, headers map[string]string) Response {
	return Response{
		status:     HTTP_OK,
		statusCode: 200,
		body:       body,
		protocol:   HTTP_1_1,
		headers:    headers,
	}
}

// toByte Converts a Response to a byte array in order to send it back to the client via tcp.Connection.Write
func (r *Response) toByte() (res []byte) {
	var str strings.Builder

	if "" != r.protocol {
		str.WriteString(r.protocol)
		str.WriteString(SPACE)
	}
	if 0 != r.statusCode {
		str.WriteString(strconv.Itoa(r.statusCode))
		str.WriteString(SPACE)
	}
	if "" != r.status {
		str.WriteString(r.status)
		str.WriteString(NEW_LINE)
	}
	if nil != r.headers {
		str.WriteString(hashmapMapToString(r.headers, func(header string, value string) string { return fmt.Sprintf("%s: %s\n", header, value) }))
		str.WriteString(NEW_LINE)
	}
	res = append(res, []byte(str.String())...)
	if nil != r.body {
		res = append(res, r.body...)
	}
	return
}

// ResponseBuilder A Response builder. This is the primarily method used for creating responses.
type ResponseBuilder struct {
	status     string
	statusCode int
	body       []byte
	protocol   string
	headers    map[string]string
}

func (r *Response) Body(body []byte) *Response {
	r.body = body
	return r
}

func (r *Response) Status(status string) *Response {
	r.status = status
	return r
}

func (r *Response) StatusCode(statusCode int) *Response {
	r.statusCode = statusCode
	return r
}

func (r *Response) Header(header, value string) *Response {
	r.headers[header] = value
	return r
}

func (r *Response) Headers(headers map[string]string) *Response {
	r.headers = headers
	return r
}

// newResponseBuilder Initializes a ResponseBuilder
func newResponseBuilder() *ResponseBuilder {
	return &ResponseBuilder{
		body:     nil,
		protocol: "HTTP/1.1",
		headers:  make(map[string]string),
	}
}

// Status Configure status for current ResponseBuilder
func (b *ResponseBuilder) Status(status string) *ResponseBuilder {
	b.status = status
	return b
}

// StatusCode Configure status code for current ResponseBuilder
func (b *ResponseBuilder) StatusCode(code int) *ResponseBuilder {
	b.statusCode = code
	return b
}

// Body Configure body for current ResponseBuilder
func (b *ResponseBuilder) Body(body []byte) *ResponseBuilder {
	b.body = body
	return b
}

// Protocol Configure protocol for current ResponseBuilder
func (b *ResponseBuilder) Protocol(proto string) *ResponseBuilder {
	b.protocol = proto
	return b
}

// Header Configure a single header for current ResponseBuilder. Can be called multiple times for different headers.
func (b *ResponseBuilder) Header(key string, value string) *ResponseBuilder {
	b.headers[key] = value
	return b
}

// Build Builds a Response with a ResponseBuilder
func (b *ResponseBuilder) Build() Response {
	return Response{
		status:     b.status,
		statusCode: b.statusCode,
		body:       b.body,
		protocol:   b.protocol,
		headers:    b.headers,
	}
}
