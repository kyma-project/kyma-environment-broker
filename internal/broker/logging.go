package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
)

var openKeys = map[string]struct{}{
	"sm_url":           {},
	"xsappname":        {},
	"globalaccount_id": {},
	"subaccount_id":    {},
}

func hideSensitiveDataFromRawContext(d []byte) map[string]interface{} {
	var data map[string]interface{}
	err := json.Unmarshal(d, &data)
	if err != nil {
		return map[string]interface{}{}
	}
	for k, v := range data {
		if v == nil {
			continue
		}
		switch reflect.TypeOf(v).Kind() {
		case reflect.String:
			if _, exists := openKeys[k]; !exists {
				data[k] = maskedKubeconfig
			}
		case reflect.Map:
			data[k] = hideSensitiveDataFromContext(v.(map[string]interface{}))
		}
	}

	return data
}

func hideSensitiveDataFromContext(input map[string]interface{}) map[string]interface{} {
	for k, v := range input {
		if v == nil {
			continue
		}
		if reflect.TypeOf(v).Kind() == reflect.String {
			if _, exists := openKeys[k]; !exists {
				input[k] = maskedKubeconfig
			}
		}
		if reflect.TypeOf(v).Kind() == reflect.Map {
			input[k] = hideSensitiveDataFromContext(v.(map[string]interface{}))
		}
	}

	return input
}

func marshallRawContext(d map[string]interface{}) string {
	b, err := json.Marshal(d)
	if err != nil {
		return "unable to marshal context data"
	}
	return string(b)
}

// strippingHandler is a slog.Handler that truncates attributes with the given keys
// to maxInstanceDetailsLen characters. Prevents brokerapi's "instance-details"
// attribute (which embeds the full raw request payload) from flooding test output.
const maxInstanceDetailsLen = 1024

type strippingHandler struct {
	inner slog.Handler
	strip map[string]struct{}
}

func NewStrippingHandler(inner slog.Handler, keys ...string) *strippingHandler {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return &strippingHandler{inner: inner, strip: m}
}

func (h *strippingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *strippingHandler) Handle(ctx context.Context, r slog.Record) error {
	filtered := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		if _, truncate := h.strip[a.Key]; truncate {
			s := fmt.Sprintf("%v", a.Value.Any())
			if len(s) > maxInstanceDetailsLen {
				s = s[:maxInstanceDetailsLen] + "...[truncated]"
			}
			filtered.AddAttrs(slog.String(a.Key, s))
		} else {
			filtered.AddAttrs(a)
		}
		return true
	})
	return h.inner.Handle(ctx, filtered)
}

func (h *strippingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	filtered := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		if _, skip := h.strip[a.Key]; !skip {
			filtered = append(filtered, a)
		} else {
			s := fmt.Sprintf("%v", a.Value.Any())
			if len(s) > maxInstanceDetailsLen {
				s = s[:maxInstanceDetailsLen] + "...[truncated]"
			}
			filtered = append(filtered, slog.String(a.Key, s))
		}
	}
	return &strippingHandler{inner: h.inner.WithAttrs(filtered), strip: h.strip}
}

func (h *strippingHandler) WithGroup(name string) slog.Handler {
	return &strippingHandler{inner: h.inner.WithGroup(name), strip: h.strip}
}
