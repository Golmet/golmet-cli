// Package jev é o cliente do classificador calibrado Typesafe/Jev
// (FORMATO §10 — kit.jev, DR-015): POST para a API com texto + critérios
// TypeSafe. SEM degradação silenciosa: key ausente ou API indisponível
// devolve resposta ESTRUTURADA com Offline=true e motivo declarado.
package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// EndpointDefault é o endpoint público do runtime Typesafe (DR-015) —
// rota /v1/systemone, payload {state, model, questions} (paridade com o
// runtime de referência: model "jev-latest" acompanha as versões servidas).
const EndpointDefault = "https://api.typesafe.ai/v1/systemone"

// modelDefault é o modelo solicitado ao runtime System One.
const modelDefault = "jev-latest"

// Endpoint resolve o endpoint da API: env TYPESAFE_API_URL com fallback
// ao default. Compartilhado pelo doctor (smoke) e pelo MCP.
func Endpoint() string {
	if u := os.Getenv("TYPESAFE_API_URL"); u != "" {
		return u
	}
	return EndpointDefault
}

// APIKey resolve a key: env TYPESAFE_API_KEY.
func APIKey() string { return os.Getenv("TYPESAFE_API_KEY") }

// Response é a resposta estruturada — Offline=true SEMPRE carrega Reason.
type Response struct {
	Offline bool            `json:"offline"`
	Reason  string          `json:"reason,omitempty"`
	Status  int             `json:"status,omitempty"`
	Raw     json.RawMessage `json:"raw,omitempty"`
}

// Classify submete (texto, critérios) ao runtime Jev. Nunca retorna erro:
// falhas viram degradação estruturada (Offline=true + Reason).
func Classify(ctx context.Context, text, questions string) *Response {
	key := APIKey()
	if key == "" {
		return &Response{Offline: true,
			Reason: "TYPESAFE_API_KEY ausente — classificação offline (degradação heurística declarada)"}
	}
	// questions chega como JSON string do tool; o runtime espera OBJETO no wire.
	var questionsJSON json.RawMessage
	if questions == "" {
		questionsJSON = json.RawMessage("{}")
	} else if !json.Valid([]byte(questions)) {
		return &Response{Offline: true,
			Reason: "critérios questions não são JSON válido — verifique o formato TypeSafe"}
	}
	questionsJSON = json.RawMessage(questions)
	body, err := json.Marshal(map[string]any{
		"state":     text,
		"model":     modelDefault,
		"questions": questionsJSON,
	})
	if err != nil {
		return &Response{Offline: true, Reason: "payload ilegível: " + err.Error()}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, Endpoint(), bytes.NewReader(body))
	if err != nil {
		return &Response{Offline: true, Reason: "endpoint inválido: " + err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &Response{Offline: true, Reason: "runtime Jev indisponível: " + err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &Response{Offline: true, Status: resp.StatusCode,
			Reason: fmt.Sprintf("runtime Jev respondeu %d — verifique a key/endpoint", resp.StatusCode)}
	}
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return &Response{Offline: true, Status: resp.StatusCode,
			Reason: "resposta ilegível do runtime Jev: " + err.Error()}
	}
	return &Response{Offline: false, Status: resp.StatusCode, Raw: raw}
}

// SmokeOutcome classifica o smoke POST do doctor (--network).
type SmokeOutcome int

const (
	SmokeOK          SmokeOutcome = iota // 2xx — runtime operacional
	SmokeRejected                        // 401/403 — key rejeitada
	SmokeUnavailable                     // rede/endpoint indisponível ou key ausente
)

// SmokeHealth executa o smoke POST opcional do doctor (--network) —
// sem payload de negócio;detail descreve o resultado.
func SmokeHealth(ctx context.Context) (SmokeOutcome, string) {
	key := APIKey()
	if key == "" {
		return SmokeUnavailable, "TYPESAFE_API_KEY ausente"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, Endpoint(),
		strings.NewReader(`{"state":"golmet doctor smoke","model":"`+modelDefault+`","questions":{"sanidade":{"type":"choice","criteria":{"ok":"runtime acessivel","nok":"runtime inacessivel"}}}}`))
	if err != nil {
		return SmokeUnavailable, "smoke não disparado (" + err.Error() + ")"
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return SmokeUnavailable, "smoke indisponível (rede?) — key presente"
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return SmokeRejected, fmt.Sprintf("smoke respondeu %d — key rejeitada", resp.StatusCode)
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return SmokeOK, fmt.Sprintf("smoke respondeu %d — runtime Jev operacional", resp.StatusCode)
	default:
		return SmokeUnavailable, fmt.Sprintf("smoke respondeu %d — verifique o endpoint", resp.StatusCode)
	}
}
