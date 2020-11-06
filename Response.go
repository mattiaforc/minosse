package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Response Response structure
type Response struct {
	status     string
	statusCode int
	body       []byte
	protocol   string
	headers    map[string]string
}

// toByte Converts a Response to a byte array in order to send it back to the client via tcp.Connection.Write
func (r *Response) toByte() (res []byte) {
	var str strings.Builder

	// TODO: nil checks...
	str.WriteString(r.protocol)
	str.WriteString(" ")
	str.WriteString(strconv.Itoa(r.statusCode))
	str.WriteString(" ")
	str.WriteString(r.status)
	str.WriteString("\n")
	str.WriteString(hashmapMapToString(r.headers, func(header string, value string) string { return fmt.Sprintf("%s: %s\n", header, value) }))
	str.WriteString("\n")
	res = append(res, []byte(str.String())...)
	res = append(res, r.body...)

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
