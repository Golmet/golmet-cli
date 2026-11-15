// Package orgs — o CATÁLOGO GLOBAL de organizações (~/.golmet/
// organizations.json, na zona global do kit — junto de skills/ e tools/).
// O invariante (FORMATO §2): o golmet.yaml do container é a FONTE DA
// VERDADE da org (estrutura + products); o registro global é o CATÁLOGO
// para o wizard (seleção de Organizações no init) — o kit nunca cria
// organização sem registrar, e organizações fora do kit entram no catálogo
// ao serem adotadas (golmet init --org sobre dir existente).
package orgs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// FileName — o catálogo na zona global do kit.
const FileName = "organizations.json"

// Entry é uma organização registrada.
type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Created string `json:"created"` // YYYY-MM-DD
}

// Catalog é o conteúdo do arquivo.
type Catalog struct {
	Organizations []Entry `json:"organizations"`
}

// Path devolve <home>/.golmet/organizations.json.
func Path(home string) string {
	return filepath.Join(home, ".golmet", FileName)
}

// Load lê o catálogo. Ausente → catálogo vazio, nil (estado inicial
// legítimo). Ilegível → erro (o caller decide — nunca corromper
// silenciosamente).
func Load(home string) (*Catalog, error) {
	data, err := os.ReadFile(Path(home))
	if os.IsNotExist(err) {
		return &Catalog{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("catálogo ilegível (%s): %w", Path(home), err)
	}
	return &c, nil
}

// CheckName — pré-checagem do nome ANTES de criar (o caller valida o
// nome da organização contra o catálogo — o kit nunca cria organização
// que não possa registrar). Mesmo nome com o MESMO path → ok (adoção/
// idempotência); mesmo nome com path DIFERENTE → erro claro.
func CheckName(home, name, path string) error {
	c, err := Load(home)
	if err != nil {
		return err
	}
	return c.checkName(name, path)
}

// checkName — a regra de unicidade do catálogo (usada por CheckName e
// Register). O nome normalizado é case-insensitive (o mesmo padrão das
// comparações de path — Windows).
func (c *Catalog) checkName(name, path string) error {
	normalized := strings.TrimSpace(name)
	for _, e := range c.Organizations {
		if !strings.EqualFold(strings.TrimSpace(e.Name), normalized) {
			continue
		}
		if samePath(e.Path, path) {
			return nil
		}
		return fmt.Errorf("já existe uma Organização com este nome registrada em %s — use outro nome ou remova a entrada do catálogo (%s)", e.Path, fileNameRef)
	}
	return nil
}

// Register grava/atualiza a entrada da org. Idempotente: mesmo nome com o
// MESMO path → entrada mantida (created original preservado); mesmo nome
// com path DIFERENTE → erro claro (CheckName; o caller pré-checa — o kit
// nunca cria organização que não possa registrar). created = hoje em
// entradas novas.
func Register(home, name, path string) error {
	c, err := Load(home)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	abs, aerr := filepath.Abs(path)
	if aerr != nil {
		return fmt.Errorf("caminho da organização inválido: %w", aerr)
	}
	if err := c.checkName(name, abs); err != nil {
		return err
	}
	for _, e := range c.Organizations {
		if strings.EqualFold(strings.TrimSpace(e.Name), name) && samePath(e.Path, abs) {
			return nil // já registrada — idempotente (created preservado)
		}
	}
	c.Organizations = append(c.Organizations, Entry{
		Name:    name,
		Path:    abs,
		Created: time.Now().Format("2006-01-02"),
	})
	return Write(home, c)
}

// fileNameRef — referência legível do catálogo nas mensagens (o caminho
// completo vem de Path(home) nos callers que o conhecem).
const fileNameRef = "~/.golmet/" + FileName

// Write persiste o catálogo (mkdir -p da zona global — a pasta já existe
// em qualquer install, mas o wizard org não exige install prévio).
func Write(home string, c *Catalog) error {
	if err := os.MkdirAll(filepath.Dir(Path(home)), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(home), append(data, '\n'), 0o644)
}

// Alive — o path da entrada existe no disco (a lista do wizard só mostra
// orgs vivas; morta = pulada com aviso).
func (e Entry) Alive() bool {
	st, err := os.Stat(e.Path)
	return err == nil && st.IsDir()
}

// samePath — comparação de paths com tolerância do SO (Windows é case-
// insensitive; o mesmo padrão do scaffold/doctor).
func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
