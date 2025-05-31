package validate

import (
	"encoding/json"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// IsNotEmpty returns true if value is not nill
func IsNotEmpty(value interface{}) bool {
	if value == nil {
		return false
	}
	if str, ok := value.(string); ok {
		return len(str) > 0
	}
	if _, ok := value.(bool); ok {
		return true
	}
	if i, ok := value.(int); ok {
		return i != 0
	}
	if i, ok := value.(uint); ok {
		return i != 0
	}
	if i, ok := value.(int8); ok {
		return i != 0
	}
	if i, ok := value.(uint8); ok {
		return i != 0
	}
	if i, ok := value.(int16); ok {
		return i != 0
	}
	if i, ok := value.(uint16); ok {
		return i != 0
	}
	if i, ok := value.(uint32); ok {
		return i != 0
	}
	if i, ok := value.(int32); ok {
		return i != 0
	}
	if i, ok := value.(int64); ok {
		return i != 0
	}
	if i, ok := value.(uint64); ok {
		return i != 0
	}
	if t, ok := value.(time.Time); ok {
		tt := time.Time{}
		return !t.IsZero() && t != tt
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Slice {
		return v.Len() > 0
	}
	return true
}

// IsNumeric check if the value contains only numbers.
func IsNumeric(value interface{}) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}
	_, err := strconv.ParseFloat(str, 64)

	return err == nil
}

// IsAlpha check if the value contains only letters (a-zA-Z). Empty string is valid.
func IsAlpha(value interface{}) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}
	return patternAlpha.MatchString(str)
}

// IsAlphanumeric check if the value contains only letters and numbers. Empty string is valid.
func IsAlphanumeric(value interface{}) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}
	return patternAlphanumeric.MatchString(str)
}

// IsAlphanumericSpace check if the value contains only letters, numbers and space. Empty string is valid.
func IsAlphanumericSpace(value interface{}) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}
	return patternAlphanumericSpace.MatchString(str)
}

// IsAlphaSpace check if the value contains only letters and space. Empty string is valid.
func IsAlphaSpace(value interface{}) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}
	return patternAlphaSpace.MatchString(str)
}

// IsEmail check if the value is an email.
func IsEmail(value interface{}) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}
	return patternEmail.MatchString(toString(value))
}

// IsLatitude check if the value is an latitude.
func IsLatitude(value interface{}) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}

	return patternLatitude.MatchString(toString(value))
}

// IsLongitude check if the value is an longitude.
func IsLongitude(value interface{}) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}

	return patternLongitude.MatchString(toString(value))
}

// IsURL check if the value is an URL.
func IsURL(value interface{}) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}
	if str == "" || len(str) >= 2083 || len(str) <= 3 || strings.HasPrefix(str, ".") {
		return false
	}
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	if strings.HasPrefix(u.Host, ".") {
		return false
	}
	if u.Host == "" && (u.Path != "" && !strings.Contains(u.Path, ".")) {
		return false
	}
	return patternURL.MatchString(str)
}

// IsJSON check if the value is valid JSON (note: uses json.Unmarshal).
func IsJSON(value interface{}) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(toString(value)), &js) == nil
}

// IsLowerThanEqual return true if value is greather than equal given number
// this will evaluate value of int, lenght of string and number of slices.
func IsLowerThanEqual(value interface{}, max interface{}) (res bool) {
	if value == nil {
		return true
	}
	return dataLength(value) <= dataLength(max)
}

// IsGreaterThanEqual return true if value is greather than equal given number
// this will evaluate value of int, lenght of string and number of slices.
func IsGreaterThanEqual(value interface{}, min interface{}) (res bool) {
	if value == nil {
		return true
	}
	return dataLength(value) >= dataLength(min)
}

// IsLowerThan return true if value is lower than given number
// this will evaluate value of int, lenght of string and number of slices.
func IsLowerThan(value interface{}, max interface{}) (res bool) {
	if value == nil {
		return true
	}
	return dataLength(value) < dataLength(max)
}

// IsGreaterThan return true if value is greather than given number
// this will evaluate value of int, lenght of string and number of slices.
func IsGreaterThan(value interface{}, min interface{}) (res bool) {
	if value == nil {
		return true
	}
	return dataLength(value) > dataLength(min)
}

// IsOnRange return true if value is greather than equal given min and lowerthan than equal given max
// this will evaluate value of int, lenght of string and number of slices.
func IsOnRange(value interface{}, min interface{}, max interface{}) bool {
	return IsGreaterThanEqual(value, min) && IsLowerThanEqual(value, max)
}

// IsContains check if the value contains the substring.
func IsContains(value interface{}, substring string) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}
	return strings.Contains(toString(value), substring)
}

// IsMatches check if value matches the pattern (pattern is regular expression)
// In case of error return false
func IsMatches(value interface{}, pattern string) bool {
	str := toString(value)
	if !IsNotEmpty(str) {
		return true
	}
	match, _ := regexp.MatchString(pattern, toString(value))
	return match
}

// IsSame check if the value is identicaly same with given param
func IsSame(value interface{}, param interface{}) bool {
	value = toString(value)
	if !IsNotEmpty(value) {
		return true
	}
	return value == toString(param)
}

// IsIn check if the value is exists in given param
func IsIn(value interface{}, param ...string) bool {
	value = toString(value)
	if !IsNotEmpty(value) {
		return true
	}
	if len(param) > 0 {
		for _, v := range param {
			if v == value {
				return true
			}
		}
	}
	return false
}

// IsNotIn check if the value is not exists in given param
func IsNotIn(value interface{}, param ...string) bool {
	value = toString(value)
	if !IsNotEmpty(value) {
		return true
	}
	if len(param) > 0 {
		for _, v := range param {
			if v == value {
				return false
			}
		}
	}
	return true
}
