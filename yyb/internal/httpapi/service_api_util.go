package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var (
	errUnexpectedData  = errors.New("unexpected api data")
	errSessionNotFound = errors.New("qr session not found")
)

type responseRecorder struct {
	status int
	body   bytes.Buffer
}

func (w *responseRecorder) Header() http.Header { return http.Header{} }

func (w *responseRecorder) Write(b []byte) (int, error) { return w.body.Write(b) }

func (w *responseRecorder) WriteHeader(statusCode int) { w.status = statusCode }

func (w *responseRecorder) decode(dst any) error {
	return json.Unmarshal(w.body.Bytes(), dst)
}

func (w *responseRecorder) apiError() error {
	var env apiEnvelope
	if err := w.decode(&env); err != nil {
		return fmt.Errorf("http %d", w.status)
	}
	if env.Msg != "" {
		return errors.New(env.Msg)
	}
	return fmt.Errorf("http %d", w.status)
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func strVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func jsonUnmarshal(b []byte, dst any) error {
	return json.Unmarshal(b, dst)
}
