package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	pgxmock "github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"
)

type argMatcher func(any) bool

func (m argMatcher) Match(v interface{}) bool {
	return m(v)
}

func newMockDB(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	return mock
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func decodeBody[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	return out
}

func ptrString(v string) *string     { return &v }
func ptrInt64(v int64) *int64        { return &v }
func ptrFloat64(v float64) *float64  { return &v }
func ptrTime(v time.Time) *time.Time { return &v }

func nilArg() argMatcher {
	return func(v any) bool {
		if v == nil {
			return true
		}
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice:
			return rv.IsNil()
		default:
			return false
		}
	}
}

func boolPtrArg(want bool) argMatcher {
	return func(v any) bool {
		got, ok := v.(*bool)
		return ok && got != nil && *got == want
	}
}

func timePtrArg(want time.Time) argMatcher {
	return func(v any) bool {
		got, ok := v.(*time.Time)
		return ok && got != nil && got.Equal(want)
	}
}
