package rest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/logistics-id/engine/validate"
)

type Context struct {
	context.Context

	Response http.ResponseWriter
	Request  *http.Request

	validator *validate.Validator
	logger    *zap.Logger
	once      sync.Once
}

// Bind decodes the JSON request body into the given struct
func (c *Context) Bind(v any) error {
	if c.Request.Method == http.MethodGet {
		// Bind from URL query params
		if err := c.bindQueryParams(v); err != nil {
			return BadRequest()
		}

		// Bind URL params if struct has any `param` tags
		if err := c.bindPathParams(v); err != nil {
			return BadRequest()
		}

		return nil
	}

	// For POST/PUT/DELETE, check if body has content before decoding
	hasBody := c.Request.ContentLength > 0

	if hasBody {
		decoder := json.NewDecoder(c.Request.Body)
		// decoder.DisallowUnknownFields()
		if err := decoder.Decode(v); err != nil {
			c.logger.Warn("Bind error", zap.Error(err))
			return BadRequest()
		}
	}

	// Bind URL params if struct has any `param` tags
	if err := c.bindPathParams(v); err != nil {
		return BadRequest()
	}

	// Only validate if we decoded a body (actions without body don't need validation)
	if err := c.Validate(v); !err.Valid {
		return err
	}

	return nil
}

func (c *Context) bindPathParams(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		paramKey := field.Tag.Get("param")
		if paramKey == "" {
			continue
		}

		paramValue := c.Param(paramKey)
		if paramValue == "" {
			continue
		}

		fv := rv.Field(i)
		if fv.CanSet() && fv.Kind() == reflect.String {
			fv.SetString(paramValue)
		}
	}

	return nil
}

func (c *Context) Validate(obj any) (resp *validate.Response) {
	c.lazyinit()

	if vr, ok := obj.(validate.Request); ok {
		resp = c.validator.Request(vr)
	} else {
		resp = c.validator.Struct(obj)
	}

	return
}

// lazyinit initialing validator instances for one of time only.
func (c *Context) lazyinit() {
	c.once.Do(func() {
		c.validator = validate.New()
	})
}

// JSON writes a standard JSON response with status code
func (c *Context) JSON(code int, data any) error {
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

// Error returns a structured error response with the given status code
func (c *Context) Error(code int, message Message, errs any) error {
	return c.JSON(code, ResponseBody{
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

func (c *Context) Respond(body any, err error) error {
	switch {
	case err == nil:
		// Determine status code: use custom if set,
		statusCode := http.StatusOK

		if rb, ok := body.(*ResponseBody); ok {
			// Use custom status code if set
			if rb.StatusCode > 0 {
				statusCode = rb.StatusCode
			}

			if rb.Message == "" {
				rb.Message = string(MsgSuccess)
			}
			rb.Success = true
			return c.JSON(statusCode, rb)
		}

		return c.JSON(statusCode, ResponseBody{
			Success: true,
			Message: string(MsgSuccess),
			Data:    body,
		})

	case errors.As(err, new(*validate.Response)):
		ve := err.(*validate.Response)
		return c.JSON(http.StatusUnprocessableEntity, ResponseBody{
			Success: false,
			Message: string(MsgValidationError),
			Errors:  ve.GetMessages(),
		})

	case errors.As(err, new(HTTPError)):
		he := err.(HTTPError)
		return c.JSON(he.Code, ResponseBody{
			Success: false,
			Message: he.Error(),
		})

	case errors.Is(err, sql.ErrNoRows):
		return c.JSON(http.StatusNotFound, ResponseBody{
			Success: false,
			Message: string(MsgNotFound),
			Errors:  err.Error(),
		})

	default:
		return c.JSON(http.StatusInternalServerError, ResponseBody{
			Success: false,
			Message: string(MsgInternalError),
			Errors:  err.Error(),
		})
	}
}

func (c *Context) bindQueryParams(v any) error {
	return bindStructFields(v, c.Request.URL.Query())
}

func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return nil
	}

	// Handle pointer types
	if field.Kind() == reflect.Ptr {
		// Get the element type
		elemType := field.Type().Elem()

		// Create a new instance of the element type
		elemValue := reflect.New(elemType).Elem()

		// Set the value on the element
		if err := setFieldValue(elemValue, value); err != nil {
			return err
		}

		// Set the pointer to point to the new value
		field.Set(elemValue.Addr())
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Uint, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(u)
	case reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)
	}
	return nil
}

func bindStructFields(v any, values url.Values) error {
	val := reflect.ValueOf(v).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if fieldType.Anonymous && field.Kind() == reflect.Struct {
			ptr := field.Addr().Interface()
			if err := bindStructFields(ptr, values); err != nil {
				return err
			}
			continue
		}

		tag := fieldType.Tag.Get("query")
		if tag == "" {
			tag = strings.ToLower(fieldType.Name)
		}

		paramVal := values.Get(tag) // ← this works because it's url.Values
		if paramVal == "" {
			continue
		}

		if err := setFieldValue(field, paramVal); err != nil {
			return fmt.Errorf("failed to bind field '%s': %w", tag, err)
		}
	}
	return nil
}
