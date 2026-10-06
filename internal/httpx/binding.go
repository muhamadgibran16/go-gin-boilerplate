package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func init() {
	// Report validation errors with the names clients use (json/form tags)
	// instead of Go field names, e.g. "perPage" rather than "PerPage".
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(fieldName)
	}
}

func fieldName(f reflect.StructField) string {
	for _, tag := range []string{"json", "form", "uri"} {
		name, _, _ := strings.Cut(f.Tag.Get(tag), ",")
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}
	return f.Name
}

// BindJSON binds and validates the JSON body into obj.
// On failure it aborts with a 400 response and returns false.
//
//	var req CreateRequest
//	if !httpx.BindJSON(c, &req) {
//		return
//	}
func BindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		Abort(c, bindError(err, "Invalid JSON body"))
		return false
	}
	return true
}

// BindQuery binds and validates the query string into obj.
// On failure it aborts with a 400 response and returns false.
func BindQuery(c *gin.Context, obj any) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		Abort(c, bindError(err, "Invalid query parameters"))
		return false
	}
	return true
}

// ParamUUID parses the path parameter name as a UUID.
// On failure it aborts with a 400 response and returns false.
func ParamUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		Abort(c, apperror.Validation(map[string]string{name: name + " must be a valid UUID"}))
		return uuid.Nil, false
	}
	return id, true
}

var errPayloadTooLarge = apperror.PayloadTooLarge("Request body too large")

// bindError converts a binding error into a client-safe application error
func bindError(err error, fallback string) error {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return errPayloadTooLarge.Wrap(err)
	}

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		fields := make(map[string]string, len(validationErrs))
		for _, e := range validationErrs {
			fields[e.Field()] = validationMessage(e)
		}
		return apperror.Validation(fields)
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field != "" {
		return apperror.Validation(map[string]string{
			typeErr.Field: fmt.Sprintf("%s must be of type %s", typeErr.Field, jsonType(typeErr.Type)),
		})
	}

	var numErr *strconv.NumError
	if errors.As(err, &numErr) {
		return apperror.BadRequest(fmt.Sprintf("%s: %q is not a valid number", fallback, numErr.Num)).Wrap(err)
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return apperror.BadRequest(fallback).Wrap(err)
	}

	return apperror.BadRequest(fallback).Wrap(err)
}

// validationMessage returns a human readable message for a failed validation rule
func validationMessage(e validator.FieldError) string {
	field, param := e.Field(), e.Param()

	switch e.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "uuid", "uuid4":
		return field + " must be a valid UUID"
	case "url", "http_url":
		return field + " must be a valid URL"
	case "numeric", "number":
		return field + " must be a number"
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, strings.ReplaceAll(param, " ", ", "))
	case "min", "gte":
		return fmt.Sprintf("%s must be at least %s", field, withUnit(param, e.Kind()))
	case "max", "lte":
		return fmt.Sprintf("%s must be at most %s", field, withUnit(param, e.Kind()))
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, withUnit(param, e.Kind()))
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, withUnit(param, e.Kind()))
	case "len":
		return fmt.Sprintf("%s must be exactly %s", field, withUnit(param, e.Kind()))
	default:
		return field + " is invalid"
	}
}

// withUnit adds "characters" for strings and "items" for collections; numbers stay plain
func withUnit(param string, kind reflect.Kind) string {
	switch kind {
	case reflect.String:
		return param + " characters long"
	case reflect.Slice, reflect.Array, reflect.Map:
		return param + " items"
	default:
		return param
	}
}

func jsonType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	default:
		return "object"
	}
}
