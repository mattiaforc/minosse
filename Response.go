package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Response struct {
	status     string
	statusCode int
	body       []byte
	protocol   string
	headers    map[string]string
}

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

type ResponseBuilder struct {
	status     string
	statusCode int
	body       []byte
	protocol   string
	headers    map[string]string
}

func newResponseBuilder() *ResponseBuilder {
	return &ResponseBuilder{
		body:     nil,
		protocol: "HTTP/1.1",
		headers:  make(map[string]string),
	}
}

func (b *ResponseBuilder) Status(status string) *ResponseBuilder {
	b.status = status
	return b
}

func (b *ResponseBuilder) StatusCode(code int) *ResponseBuilder {
	b.statusCode = code
	return b
}

func (b *ResponseBuilder) Body(body []byte) *ResponseBuilder {
	b.body = body
	return b
}

func (b *ResponseBuilder) Protocol(proto string) *ResponseBuilder {
	b.protocol = proto
	return b
}

func (b *ResponseBuilder) Header(key string, value string) *ResponseBuilder {
	b.headers[key] = value
	return b
}

func (b *ResponseBuilder) Build() Response {
	return Response{
		status:     b.status,
		statusCode: b.statusCode,
		body:       b.body,
		protocol:   b.protocol,
		headers:    b.headers,
	}
}
