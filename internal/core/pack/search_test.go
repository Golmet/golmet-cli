package pack

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// searchFixture devolve um payload GitHub com 2 packs do tópico e 1 fora
// dele (para o filtro client-side).
func searchFixture() string {
	return `{"total_count":3,"items":[
		{"name":"golmet-core","full_name":"Golmet/golmet-core",
		 "owner":{"login":"victor-souza-dev"},
		 "html_url":"https://github.com/Golmet/golmet-core",
		 "stargazers_count":42,"description":"Kit Golmet",
		 "pushed_at":"2026-01-15T10:30:00Z","topics":["golmet","golmet-pack"]},
		{"name":"fora-do-topico","full_name":"alguem/fora-do-topico",
		 "owner":{"login":"alguem"},
		 "html_url":"https://github.com/alguem/fora-do-topico",
		 "stargazers_count":900,"description":"não deveria aparecer",
		 "pushed_at":"2026-02-01T00:00:00Z","topics":["outro"]},
		{"name":"gm-pack-angular","full_name":"golmet/gm-pack-angular",
		 "owner":{"login":"golmet"},
		 "html_url":"https://github.com/golmet/gm-pack-angular",
		 "stargazers_count":7,"description":"Playbook Angular",
		 "pushed_at":"2026-03-20T08:00:00Z","topics":["golmet-pack"]}
	]}`
}

func TestSearchQueryTopicAndParse(t *testing.T) {
	var gotURL *url.URL
	var gotUA, gotAccept, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL, gotUA = r.URL, r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, searchFixture())
	}))
	defer srv.Close()

	results, err := Search("bloco", PackSearchOptions{
		BaseURL: srv.URL,
		GHToken: func() (string, error) {
			t.Error("gh não deveria ser consultado sem rate limit")
			return "", nil
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if q := gotURL.Query().Get("q"); q != "bloco topic:golmet-pack" {
		t.Errorf("q = %q, want %q", q, "bloco topic:golmet-pack")
	}
	if pp := gotURL.Query().Get("per_page"); pp != "10" {
		t.Errorf("per_page = %q, want 10", pp)
	}
	if gotURL.Query().Get("sort") != "" {
		t.Errorf("sort deveria ficar a cargo do best-match com termo")
	}
	if gotUA != "golmet-cli" || gotAuth != "" {
		t.Errorf("headers inesperados: UA=%q auth=%q", gotUA, gotAuth)
	}
	if gotAccept != "application/vnd.github+json" {
		t.Errorf("Accept = %q", gotAccept)
	}
	// filtro topic: o item sem golmet-pack deve ter sido descartado
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2 (item fora do tópico filtrado)", len(results))
	}
	first := results[0]
	if first.Name != "golmet-core" || first.FullName != "Golmet/golmet-core" ||
		first.URL != "https://github.com/Golmet/golmet-core" ||
		first.Stars != 42 || first.Description != "Kit Golmet" || first.PushedAt != "2026-01-15T10:30:00Z" {
		t.Errorf("parse inesperado: %+v", first)
	}
}

func TestSearchJSONContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, searchFixture())
	}))
	defer srv.Close()
	results, err := Search("", PackSearchOptions{BaseURL: srv.URL, GHToken: func() (string, error) { return "", nil }})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	data, err := json.Marshal(results)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, item := range decoded {
		for _, key := range []string{"name", "full_name", "url", "stars", "description", "pushed_at"} {
			if _, ok := item[key]; !ok {
				t.Errorf("campo %q ausente do contrato --json", key)
			}
		}
	}
}

func TestSearchEmptyTermSortsByStars(t *testing.T) {
	var gotURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		io.WriteString(w, `{"items":[]}`)
	}))
	defer srv.Close()

	if _, err := Search("", PackSearchOptions{BaseURL: srv.URL, GHToken: func() (string, error) { return "", nil }}); err != nil {
		t.Fatalf("search sem termo: %v", err)
	}
	if q := gotURL.Query().Get("q"); q != "topic:golmet-pack" {
		t.Errorf("q = %q, want %q", q, "topic:golmet-pack")
	}
	if gotURL.Query().Get("sort") != "stars" || gotURL.Query().Get("order") != "desc" {
		t.Errorf("termo vazio deveria ordenar por stars: %s", gotURL.RawQuery)
	}
}

func TestSearchLimitClampsPerPage(t *testing.T) {
	var gotPer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPer = r.URL.Query().Get("per_page")
		io.WriteString(w, `{"items":[]}`)
	}))
	defer srv.Close()
	if _, err := Search("x", PackSearchOptions{BaseURL: srv.URL, Limit: 3, GHToken: func() (string, error) { return "", nil }}); err != nil {
		t.Fatal(err)
	}
	if gotPer != "3" {
		t.Errorf("per_page = %q, want 3", gotPer)
	}
}

func TestSearch403WithoutGHHintsAuth(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		http.Error(w, "API rate limit exceeded", http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := Search("x", PackSearchOptions{
		BaseURL: srv.URL,
		GHToken: func() (string, error) { return "", errors.New("gh ausente") },
	})
	if err == nil {
		t.Fatal("403 sem gh deveria falhar")
	}
	if !strings.Contains(err.Error(), "rate limit") || !strings.Contains(err.Error(), "gh auth") {
		t.Errorf("mensagem sem dica de auth: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (sem token, não repete)", calls)
	}
}

func TestSearch403RetriesWithGHToken(t *testing.T) {
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auths = append(auths, r.Header.Get("Authorization"))
		if len(auths) == 1 {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, "rate limit")
			return
		}
		io.WriteString(w, searchFixture())
	}))
	defer srv.Close()

	results, err := Search("x", PackSearchOptions{
		BaseURL: srv.URL,
		GHToken: func() (string, error) { return "tok-secreto", nil },
	})
	if err != nil {
		t.Fatalf("search com retry: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if len(auths) != 2 || auths[0] != "" || auths[1] != "Bearer tok-secreto" {
		t.Errorf("auths = %v, want [\"\" \"Bearer tok-secreto\"]", auths)
	}
}

func TestSearch403WithTokenStillLimited(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := Search("x", PackSearchOptions{
		BaseURL: srv.URL,
		GHToken: func() (string, error) { return "tok", nil },
	})
	if err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("429 com token também limitado deveria falhar com rate limit: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (anônimo + token)", calls)
	}
}

func TestSearchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	_, err := Search("x", PackSearchOptions{
		BaseURL:    srv.URL,
		HTTPClient: &http.Client{Timeout: 5 * time.Millisecond},
	})
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("erro de timeout esperado, veio: %v", err)
	}
}

func TestSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "boom interno\nlinha 2")
	}))
	defer srv.Close()

	_, err := Search("x", PackSearchOptions{BaseURL: srv.URL, GHToken: func() (string, error) { return "", nil }})
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") || !strings.Contains(err.Error(), "boom interno") {
		t.Fatalf("erro HTTP esperado com causa curta, veio: %v", err)
	}
}

func TestRenderSearchResults(t *testing.T) {
	results := []RepoResult{
		{Name: "gm-pack-angular", FullName: "golmet/gm-pack-angular",
			URL: "https://github.com/golmet/gm-pack-angular", Stars: 7,
			Description: "Playbook Angular", PushedAt: "2026-03-20T08:00:00Z"},
		{Name: "golmet-core", FullName: "Golmet/golmet-core",
			URL: "https://github.com/Golmet/golmet-core", Stars: 120,
			Description: "Kit Golmet", PushedAt: "2026-01-15T10:30:00Z"},
	}
	var out strings.Builder
	RenderSearchResults(&out, results)
	got := out.String()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("linhas = %d, want 3 (2 resultados + dica):\n%s", len(lines), got)
	}
	if !strings.Contains(lines[0], "gm-pack-angular@golmet — ★  7 — Playbook Angular (atualizado 2026-03-20)") {
		t.Errorf("linha 1 sem alinhamento/data: %q", lines[0])
	}
	if !strings.Contains(lines[1], "golmet-core@Golmet — ★120 — Kit Golmet (atualizado 2026-01-15)") {
		t.Errorf("linha 2 inesperada: %q", lines[1])
	}
	if !strings.Contains(lines[2], "golmet pack add https://github.com/<owner>/<repo> --engine <id>") {
		t.Errorf("dica final ausente: %q", lines[2])
	}
}

func TestRenderSearchResultsEmptyAndWithoutDescription(t *testing.T) {
	var out strings.Builder
	RenderSearchResults(&out, nil)
	if !strings.Contains(out.String(), "nenhum pack encontrado") {
		t.Errorf("lista vazia sem mensagem: %q", out.String())
	}
	out.Reset()
	RenderSearchResults(&out, []RepoResult{
		{Name: "x", FullName: "org/x", URL: "https://github.com/org/x",
			Stars: 1, PushedAt: "2026-05-05T00:00:00Z"},
	})
	got := out.String()
	if !strings.Contains(got, "x@org — ★1 (atualizado 2026-05-05)") {
		t.Errorf("linha sem description inesperada: %q", got)
	}
	if !strings.Contains(got, "dica: golmet pack add") {
		t.Errorf("dica final esperada com resultado: %q", got)
	}
}
