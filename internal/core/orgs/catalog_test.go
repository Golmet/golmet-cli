package orgs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- catálogo global de organizações (~/.golmet/organizations.json) ----

func TestCatalogRegisterAndLoad(t *testing.T) {
	home := t.TempDir()
	if err := Register(home, "MinhaOrg", filepath.Join(home, "raiz", "Org")); err != nil {
		t.Fatalf("register: %v", err)
	}
	c, err := Load(home)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(c.Organizations) != 1 {
		t.Fatalf("entries = %d, want 1: %+v", len(c.Organizations), c.Organizations)
	}
	e := c.Organizations[0]
	if e.Name != "MinhaOrg" || !strings.HasSuffix(e.Path, "Org") {
		t.Errorf("entry inesperada: %+v", e)
	}
	if matched, _ := filepath.Match("????-??-??", e.Created); !matched {
		t.Errorf("created = %q, want YYYY-MM-DD", e.Created)
	}
	// o arquivo existe na zona global do kit, no contrato declarado
	if _, serr := os.Stat(filepath.Join(home, ".golmet", FileName)); serr != nil {
		t.Errorf("catálogo não gravado: %v", serr)
	}
}

// Idempotência: mesma org + MESMO path → uma entrada só (created original
// preservado).
func TestCatalogRegisterIdempotent(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "Org")
	if err := Register(home, "MinhaOrg", p); err != nil {
		t.Fatal(err)
	}
	c1, _ := Load(home)
	created := c1.Organizations[0].Created
	if err := Register(home, "MinhaOrg", p); err != nil {
		t.Fatalf("re-register deveria ser idempotente: %v", err)
	}
	c2, _ := Load(home)
	if len(c2.Organizations) != 1 {
		t.Fatalf("entries = %d, want 1 (idempotente)", len(c2.Organizations))
	}
	if c2.Organizations[0].Created != created {
		t.Errorf("created alterado na re-registro: %s → %s", created, c2.Organizations[0].Created)
	}
}

// Nome duplicado com path DIFERENTE → erro claro, catálogo intacto.
func TestCatalogDuplicateNameDifferentPath(t *testing.T) {
	home := t.TempDir()
	if err := Register(home, "MinhaOrg", filepath.Join(home, "antiga")); err != nil {
		t.Fatal(err)
	}
	err := Register(home, "MinhaOrg", filepath.Join(home, "outra"))
	if err == nil {
		t.Fatalf("nome duplicado com path diferente deveria falhar")
	}
	for _, want := range []string{"já existe uma Organização", filepath.Join(home, "antiga"), "use outro nome"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("erro sem %q: %v", want, err)
		}
	}
	c, _ := Load(home)
	if len(c.Organizations) != 1 {
		t.Errorf("catálogo alterado no erro: %+v", c.Organizations)
	}
	// CheckName (pré-checagem ANTES de criar — o kit nunca cria org que
	// não possa registrar) reporta o mesmo conflito
	if cerr := CheckName(home, "MinhaOrg", filepath.Join(home, "outra")); cerr == nil {
		t.Errorf("CheckName deveria reportar o conflito")
	}
	if cerr := CheckName(home, "MinhaOrg", filepath.Join(home, "antiga")); cerr != nil {
		t.Errorf("CheckName mesmo path deveria passar (adoção): %v", cerr)
	}
}

// Catálogo ausente → vazio (estado inicial legítimo); ilegível → erro.
func TestCatalogLoadEmptyAndBroken(t *testing.T) {
	home := t.TempDir()
	c, err := Load(home)
	if err != nil || len(c.Organizations) != 0 {
		t.Fatalf("catálogo ausente = vazio; got %v / %v", c, err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".golmet"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".golmet", FileName), []byte("{quebrado"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(home); err == nil {
		t.Errorf("catálogo ilegível deveria falhar (nunca corromper silenciosamente)")
	}
}

// Alive — path vivo listável; path morto pulado pelo wizard.
func TestCatalogAlive(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "Org")
	os.MkdirAll(dir, 0o755)
	if !(Entry{Name: "viva", Path: dir}).Alive() {
		t.Errorf("dir vivo deveria ser Alive")
	}
	if (Entry{Name: "morta", Path: filepath.Join(home, "sumiu")}).Alive() {
		t.Errorf("path morto não deveria ser Alive")
	}
	// arquivo (não dir) não é Alive
	file := filepath.Join(home, "arq.txt")
	os.WriteFile(file, []byte("x"), 0o644)
	if (Entry{Name: "arquivo", Path: file}).Alive() {
		t.Errorf("arquivo não é organização")
	}
}

// O JSON no disco carrega o contrato declarado (organizations/name/path/created).
func TestCatalogJSONShape(t *testing.T) {
	home := t.TempDir()
	if err := Register(home, "OrgX", filepath.Join(home, "raiz")); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".golmet", FileName))
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json: %v", err)
	}
	if _, ok := raw["organizations"]; !ok {
		t.Errorf("json sem a chave organizations: %s", data)
	}
	var entries []map[string]any
	json.Unmarshal(raw["organizations"], &entries)
	for _, k := range []string{"name", "path", "created"} {
		if _, ok := entries[0][k]; !ok {
			t.Errorf("entry sem %q: %s", k, entries[0])
		}
	}
}
