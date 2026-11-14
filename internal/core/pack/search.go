package pack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Descoberta descentralizada de packs: não há registry central. Packs
// públicos do Golmet publicam no GitHub com o tópico SearchTopic e o search
// consulta a Search API do GitHub — anonimamente por padrão; em 403/429 usa
// o token do CLI gh autenticado (quando disponível) para subir a cota.

const (
	// SearchTopic é a convenção da comunidade: todo pack público do Golmet
	// publica no GitHub com este tópico.
	SearchTopic = "golmet-pack"
	// SearchBaseURLDefault é a API pública do GitHub.
	SearchBaseURLDefault = "https://api.github.com"
	// SearchLimitDefault é o máximo de resultados por busca (per_page).
	SearchLimitDefault = 10
	searchTimeout      = 10 * time.Second
)

// RepoResult é um pack público encontrado pelo search. As tags JSON são o
// contrato de saída do `golmet pack search --json`.
type RepoResult struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	URL         string `json:"url"`
	Stars       int    `json:"stars"`
	Description string `json:"description"`
	PushedAt    string `json:"pushed_at"`
}

// PackSearchOptions controla a busca — campos injetáveis para testes com
// httptest. Zero value busca o tópico SearchTopic anonimamente com timeout
// de 10s.
type PackSearchOptions struct {
	BaseURL    string                 // default SearchBaseURLDefault
	HTTPClient *http.Client           // default com timeout de 10s
	Limit      int                    // default SearchLimitDefault
	GHToken    func() (string, error) // default: gh auth token (lookPath + exec, nunca lê arquivos)
}

func (o *PackSearchOptions) fill() {
	if o.BaseURL == "" {
		o.BaseURL = SearchBaseURLDefault
	}
	if o.HTTPClient == nil {
		o.HTTPClient = &http.Client{Timeout: searchTimeout}
	}
	if o.Limit <= 0 || o.Limit > 100 {
		o.Limit = SearchLimitDefault
	}
	if o.GHToken == nil {
		o.GHToken = ghAuthToken
	}
}

// ghAuthToken obtém o token do gh autenticado — só lookPath + exec, nunca
// lê arquivos de configuração.
func ghAuthToken() (string, error) {
	bin, err := exec.LookPath("gh")
	if err != nil {
		return "", err
	}
	out, err := exec.Command(bin, "auth", "token").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// rateLimitError marca 403/429 do GitHub para o retry autenticado.
type rateLimitError struct{ code int }

func (e *rateLimitError) Error() string {
	return fmt.Sprintf("rate limit do GitHub atingido (HTTP %d)", e.code)
}

type searchRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	HTMLURL     string   `json:"html_url"`
	Stars       int      `json:"stargazers_count"`
	Description string   `json:"description"`
	PushedAt    string   `json:"pushed_at"`
	Topics      []string `json:"topics"`
}

type searchResponse struct {
	Items []searchRepo `json:"items"`
}

// Search busca packs públicos no tópico SearchTopic. Termo vazio lista os
// mais estrelados do tópico (sort=stars). Em 403/429 tenta uma única vez
// com o token do gh autenticado; sem token disponível, devolve erro com a
// dica de autenticação.
func Search(term string, opts PackSearchOptions) ([]RepoResult, error) {
	opts.fill()
	results, err := searchOnce(term, opts, "")
	if err == nil {
		return results, nil
	}
	var rl *rateLimitError
	if !errors.As(err, &rl) {
		return nil, err
	}
	token, terr := opts.GHToken()
	if terr != nil || token == "" {
		return nil, fmt.Errorf("%w — dica: autentique o CLI gh (gh auth login) para cota maior da Search API", err)
	}
	results, err = searchOnce(term, opts, "Bearer "+token)
	if err == nil {
		return results, nil
	}
	if errors.As(err, &rl) {
		return nil, fmt.Errorf("%w — o token do gh também esgotou a cota; tente novamente mais tarde", err)
	}
	return nil, err
}

func searchOnce(term string, opts PackSearchOptions, auth string) ([]RepoResult, error) {
	q := "topic:" + SearchTopic
	term = strings.TrimSpace(term)
	if term != "" {
		q = term + " " + q
	}
	endpoint := fmt.Sprintf("%s/search/repositories?q=%s&per_page=%d",
		strings.TrimSuffix(opts.BaseURL, "/"), url.QueryEscape(q), opts.Limit)
	if term == "" {
		endpoint += "&sort=stars&order=desc"
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "golmet-cli")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := opts.HTTPClient.Do(req)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return nil, fmt.Errorf("timeout na consulta ao GitHub (%s)", opts.HTTPClient.Timeout)
		}
		return nil, fmt.Errorf("falha na consulta ao GitHub: %s", firstLine(err.Error()))
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests:
		return nil, &rateLimitError{code: resp.StatusCode}
	case resp.StatusCode != http.StatusOK:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("GitHub search HTTP %d: %s", resp.StatusCode, firstLine(string(body)))
	}
	var payload searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("resposta ilegível do GitHub: %s", firstLine(err.Error()))
	}
	results := make([]RepoResult, 0, len(payload.Items))
	for _, item := range payload.Items {
		// A query já filtra server-side; o filtro client-side é defesa
		// contra payloads sem a correspondência do tópico.
		if len(item.Topics) > 0 && !slices.Contains(item.Topics, SearchTopic) {
			continue
		}
		name := item.Name
		if name == "" && item.FullName != "" {
			name = item.FullName[strings.LastIndex(item.FullName, "/")+1:]
		}
		full := item.FullName
		if full == "" && item.Owner.Login != "" {
			full = item.Owner.Login + "/" + name
		}
		target := item.HTMLURL
		if target == "" && full != "" {
			target = "https://github.com/" + full
		}
		results = append(results, RepoResult{
			Name: name, FullName: full, URL: target,
			Stars: item.Stars, Description: item.Description, PushedAt: item.PushedAt,
		})
	}
	return results, nil
}

// RenderSearchResults imprime a lista textual alinhada com a dica final do
// pack add. A coluna de estrelas é alinhada pelo maior count.
func RenderSearchResults(w io.Writer, results []RepoResult) {
	if len(results) == 0 {
		fmt.Fprintln(w, "nenhum pack encontrado no tópico "+SearchTopic)
		return
	}
	width := 0
	for _, r := range results {
		if n := len(strconv.Itoa(r.Stars)); n > width {
			width = n
		}
	}
	for _, r := range results {
		name, owner := splitFullName(r.FullName)
		date := formatDate(r.PushedAt)
		if r.Description == "" {
			fmt.Fprintf(w, "%s@%s — ★%*d (atualizado %s)\n", name, owner, width, r.Stars, date)
			continue
		}
		fmt.Fprintf(w, "%s@%s — ★%*d — %s (atualizado %s)\n", name, owner, width, r.Stars, r.Description, date)
	}
	fmt.Fprintln(w, "dica: golmet pack add https://github.com/<owner>/<repo> --engine <id>")
}

// splitFullName deriva nome e owner de "owner/name"; sem o separador, o
// valor inteiro é tratado como nome.
func splitFullName(full string) (name, owner string) {
	ownerPart, namePart, ok := strings.Cut(full, "/")
	if !ok {
		return full, ""
	}
	return namePart, ownerPart
}

func formatDate(pushedAt string) string {
	if ts, err := time.Parse(time.RFC3339, pushedAt); err == nil {
		return ts.Format("2006-01-02")
	}
	if len(pushedAt) >= 10 {
		return pushedAt[:10]
	}
	return pushedAt
}

func firstLine(s string) string {
	if line, _, ok := strings.Cut(s, "\n"); ok {
		return strings.TrimSpace(line)
	}
	return strings.TrimSpace(s)
}
