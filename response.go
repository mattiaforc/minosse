package main

import (
	"strconv"
	"strings"
)

// Response Response structure
type Response struct {
	status     string
	statusCode int
	body       []byte
	protocol   string
	headers    string
}

func ResponseMethodNotAllowed() Response {
	return Response{
		status:     HTTP_NOT_FOUND,
		statusCode: 405,
		protocol:   HTTP_1_1,
		body:       []byte(HTTP_NOT_FOUND_BODY),
		headers:    HashmapMapToString(map[string]string{HEADER_CONTENT_TYPE: "text/plain; charset=utf-8"}, HeaderMapToString),
	}
}

func ResponseInternalServerError() Response {
	return Response{
		status:     HTTP_INTERNAL_SERVER_ERROR,
		statusCode: 500,
		body:       []byte(HTTP_INTERNAL_SERVER_ERROR),
		protocol:   HTTP_1_1,
		headers:    HashmapMapToString(map[string]string{HEADER_CONTENT_TYPE: "text/plain; charset=utf-8"}, HeaderMapToString),
	}
}

func ResponseNotFound() Response {
	return Response{
		status:     HTTP_NOT_FOUND,
		statusCode: 404,
		body:       []byte(HTTP_NOT_FOUND_BODY),
		protocol:   HTTP_1_1,
		headers:    HashmapMapToString(map[string]string{HEADER_CONTENT_TYPE: "text/plain; charset=utf-8"}, HeaderMapToString),
	}
}

func ResponseOkNoBody(headers map[string]string) Response {
	return Response{
		status:     HTTP_OK,
		statusCode: 200,
		protocol:   HTTP_1_1,
		headers:    HashmapMapToString(headers, HeaderMapToString),
	}
}

func ResponseOk(body []byte, headers map[string]string) Response {
	return Response{
		status:     HTTP_OK,
		statusCode: 200,
		body:       body,
		protocol:   HTTP_1_1,
		headers:    HashmapMapToString(headers, HeaderMapToString),
	}
}

func HeaderMapToString(header, value string) string {
	var sb strings.Builder

	sb.WriteString(header)
	sb.WriteString(": ")
	sb.WriteString(value)
	sb.WriteString(EOL)

	return sb.String()
}

func (r *Response) ResponseToByteNoBody() (res []byte) {
	var str strings.Builder

	if r.protocol != "" {
		str.WriteString(r.protocol)
		str.WriteString(SPACE)
	}
	if r.statusCode != 0 {
		str.WriteString(strconv.Itoa(r.statusCode))
		str.WriteString(SPACE)
	}
	if r.status != "" {
		str.WriteString(r.status)
		str.WriteString(EOL)
	}
	if r.headers != "" {
		str.WriteString(r.headers)
		str.WriteString(EOL)
	}
	res = append(res, []byte(str.String())...)
	return
}

// ToByte Converts a Response to a byte array in order to send it back to the client via tcp.Connection.Write
func (r *Response) ToByte() (res []byte) {
	res = append(res, r.ResponseToByteNoBody()...)
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

/*
func (r *Response) Header(header, value string) *Response {
		r.headers[header] = value
		return r
}
*/

func (r *Response) Headers(headers map[string]string) *Response {
	r.headers = HashmapMapToString(headers, HeaderMapToString)
	return r
}

// Status configure status for current ResponseBuilder
func (b *ResponseBuilder) Status(status string) *ResponseBuilder {
	b.status = status
	return b
}

// StatusCode configure status code for current ResponseBuilder
func (b *ResponseBuilder) StatusCode(code int) *ResponseBuilder {
	b.statusCode = code
	return b
}

// Body configure body for current ResponseBuilder
func (b *ResponseBuilder) Body(body []byte) *ResponseBuilder {
	b.body = body
	return b
}

// Protocol configure protocol for current ResponseBuilder
func (b *ResponseBuilder) Protocol(proto string) *ResponseBuilder {
	b.protocol = proto
	return b
}

// Header configure a single header for current ResponseBuilder. Can be called multiple times for different headers.
func (b *ResponseBuilder) Header(key, value string) *ResponseBuilder {
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
		headers:    HashmapMapToString(b.headers, HeaderMapToString),
	}
}
