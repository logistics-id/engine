package rest

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type Context struct {
	Response http.ResponseWriter
	Request  *http.Request
}

// Bind decodes the JSON request body into the given struct
func (c *Context) Bind(v interface{}) error {
	if c.Request.Body == nil {
		return BadRequest("Empty request body")
	}

	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return BadRequest("Invalid request body: " + err.Error())
	}

	return nil
}

// JSON writes a standard JSON response with status code
func (c *Context) JSON(code int, data interface{}) error {
	c.Response.Header().Set("Content-Type", "application/json")
	c.Response.WriteHeader(code)
	return json.NewEncoder(c.Response).Encode(data)
}

// Text writes a plain-text response
func (c *Context) Text(code int, msg string) {
	c.Response.Header().Set("Content-Type", "text/plain")
	c.Response.WriteHeader(code)
	c.Response.Write([]byte(msg))
}

// Success returns a standard 200 OK JSON response
func (c *Context) Success(data any, message Message) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: string(message),
		Data:    data,
	})
}

// Created returns a standard 201 Created JSON response
func (c *Context) Created(data any, message Message) error {
	return c.JSON(http.StatusCreated, Response{
		Success: true,
		Message: string(message),
		Data:    data,
	})
}

// Error returns a structured error response with the given status code
func (c *Context) Error(code int, message Message, errs any) error {
	return c.JSON(code, Response{
		Success: false,
		Message: string(message),
		Errors:  errs,
	})
}

// NoContent returns a 204 No Content response
func (c *Context) NoContent() error {
	c.Response.WriteHeader(http.StatusNoContent)
	return nil
}

// Query returns a query string parameter by key
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// Param returns a path parameter by key (using gorilla/mux)
func (c *Context) Param(key string) string {
	vars := mux.Vars(c.Request)
	return vars[key]
}

// HandlerFunc defines the function signature for route handlers
type HandlerFunc func(*Context) error

// HTTPError is a reusable error type for structured error responses
type HTTPError struct {
	Code    int
	Message string
}

func (e HTTPError) Error() string {
	return e.Message
}

// Common HTTP error helpers
func BadRequest(msg string) HTTPError {
	return HTTPError{Code: http.StatusBadRequest, Message: msg}
}

func Unauthorized(msg string) HTTPError {
	return HTTPError{Code: http.StatusUnauthorized, Message: msg}
}

func Forbidden(msg string) HTTPError {
	return HTTPError{Code: http.StatusForbidden, Message: msg}
}

func InternalServer(msg string) HTTPError {
	return HTTPError{Code: http.StatusInternalServerError, Message: msg}
}
