package jev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClassifyWithoutKeyIsStructuredOffline(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	resp := Classify(context.Background(), "texto", `{"a": "b"}`)
	if !resp.Offline {
		t.Errorf("sem key deveria ser offline")
	}
	if !strings.Contains(resp.Reason, "TYPESAFE_API_KEY") {
		t.Errorf("reason deveria declarar a key ausente: %q", resp.Reason)
	}
	if resp.Raw != nil {
		t.Errorf("offline não deveria carregar raw")
	}
}

func TestClassifyWithMockOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("esperava POST, veio %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("authorization ausente")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("payload não decodifica: %v", err)
		}
		if body["state"] == "" {
			t.Errorf("payload esperava state (campo do texto): %+v", body)
		}
		if body["model"] != modelDefault {
			t.Errorf("payload esperava model %q: %+v", modelDefault, body)
		}
		q, ok := body["questions"].(map[string]any)
		if !ok || len(q) == 0 {
			t.Errorf("questions deveria ser OBJETO decodificado: %+v", body)
		}
		w.Write([]byte(`{"resposta":"confirmada","p":0.83}`))
	}))
	defer srv.Close()
	t.Setenv("TYPESAFE_API_KEY", "test-key")
	t.Setenv("TYPESAFE_API_URL", srv.URL)

	resp := Classify(context.Background(), "RN-001 corpo", `{"confirmada": "critério"}`)
	if resp.Offline {
		t.Errorf("mock 200 deveria ser online: %+v", resp)
	}
	if resp.Status != http.StatusOK || !strings.Contains(string(resp.Raw), "confirmada") {
		t.Errorf("raw/status inesperados: %+v", resp)
	}
}

func TestClassifyDefaultEndpointIsSystemone(t *testing.T) {
	t.Setenv("TYPESAFE_API_URL", "")
	if got, want := Endpoint(), "https://api.typesafe.ai/v1/systemone"; got != want {
		t.Errorf("endpoint default = %q, esperava %q (paridade com o runtime de referência)", got, want)
	}
}

func TestClassifyWithInvalidQuestionsIsStructuredOffline(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "test-key")
	resp := Classify(context.Background(), "x", `isto não é json`)
	if !resp.Offline || resp.Status != 0 {
		t.Errorf("questions inválido deveria degradar offline sem status HTTP: %+v", resp)
	}
	if !strings.Contains(resp.Reason, "questions") {
		t.Errorf("reason deveria citar questions: %q", resp.Reason)
	}
}

func TestClassifyWithMockServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("TYPESAFE_API_KEY", "test-key")
	t.Setenv("TYPESAFE_API_URL", srv.URL)

	resp := Classify(context.Background(), "x", "{}")
	if !resp.Offline || resp.Status != http.StatusInternalServerError {
		t.Errorf("500 deveria degradar estruturado com status: %+v", resp)
	}
	if !strings.Contains(resp.Reason, "500") {
		t.Errorf("reason deveria citar o status: %q", resp.Reason)
	}
}

func TestSmokeHealthWithoutKey(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	out, detail := SmokeHealth(context.Background())
	if out != SmokeUnavailable || !strings.Contains(detail, "TYPESAFE_API_KEY") {
		t.Errorf("smoke sem key = %v (%s)", out, detail)
	}
}

func TestSmokeHealthWithMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("TYPESAFE_API_KEY", "test-key")
	t.Setenv("TYPESAFE_API_URL", srv.URL)
	if out, detail := SmokeHealth(context.Background()); out != SmokeOK {
		t.Errorf("smoke 200 = %v (%s)", out, detail)
	}

	srv401 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv401.Close()
	t.Setenv("TYPESAFE_API_URL", srv401.URL)
	if out, detail := SmokeHealth(context.Background()); out != SmokeRejected {
		t.Errorf("smoke 401 = %v (%s)", out, detail)
	}
}
