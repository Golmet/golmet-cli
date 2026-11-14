package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture: snapshot + DATA_MODEL cobrindo os casos de um corpus de validação
func writeFixture(t *testing.T) (workspace, repo string) {
	t.Helper()
	root := t.TempDir()
	repo = filepath.Join(root, "repo")
	ws := filepath.Join(root, "ws")
	mig := filepath.Join(repo, "MeuApp.Infrastructure", "Migrations")
	if err := os.MkdirAll(mig, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	snapshot := `modelBuilder.Entity("App.Agendamento", b => { b.ToTable("Agendamentos"); });
modelBuilder.Entity("App.ListaEspera", b => { b.ToTable("ListasEspera"); });
modelBuilder.Entity("App.BloqueioAgenda", b => { b.ToTable("BloqueiosAgenda"); });
modelBuilder.Entity("App.ProfissionalEspecial", b => { b.ToTable("ProfissionaisEspeciais"); });
modelBuilder.Entity("App.Usuario", b => { b.ToTable("Usuarios"); });
modelBuilder.Entity("Microsoft.AspNetCore.Identity.IdentityUser", b => { b.ToTable("AspNetUsers"); });
modelBuilder.Entity("App.ProdutoSemDoc", b => { b.ToTable("ProdutoSemDoc"); });
`
	if err := os.WriteFile(filepath.Join(mig, "AppDbContextModelSnapshot.cs"), []byte(snapshot), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"20260101_InitialCreate.cs", "20260102_AddTabela.cs", "AppDbContextModelSnapshot.Designer.cs"} {
		if err := os.WriteFile(filepath.Join(mig, name), []byte("// mig"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dm := "### Tabela `Agendamentos`\n### Tabela `ListasEsperas`\n### Tabela `BloqueiosAgendas`\n### Tabela `ProfissionaisEspeciais`\n### Tabela `Usuarios`\n### Tabela `ValesFakes`\n"
	if err := os.WriteFile(filepath.Join(ws, "specs", "DATA_MODEL.md"), []byte(dm), 0o644); err != nil {
		t.Fatal(err)
	}
	return ws, repo
}

func TestDriftClassification(t *testing.T) {
	ws, repo := writeFixture(t)

	// Diagnóstico: o arquivo que o Drift vai ler (mesmo path do internal)
	snapPath := filepath.Join(repo, "MeuApp.Infrastructure", "Migrations", "AppDbContextModelSnapshot.cs")
	ents, infr, _ := extractSnapshotEntities(snapPath)
	t.Logf("DEBUG pré-Drift extração direta: entities=%v infra=%v", ents, infr)

	rep, err := Drift(ws, repo)
	if err != nil {
		t.Fatalf("drift: %v", err)
	}
	t.Logf("DEBUG rep: dm=%v snap=%v infra=%v matched=%v", rep.DocumentedTables, rep.SnapshotEntities, rep.InfraEntities, rep.Matched)

	// extração DIRETA do mesmo arquivo que o Drift usou (diagnóstico)
	ents2, infr2, _ := extractSnapshotEntities(snapPath)
	t.Logf("DEBUG extração direta: entities=%v infra=%v", ents2, infr2)

	if rep.MigrationsCount != 2 {
		t.Errorf("migrations count = %d, want 2 (Designer/Snapshot excluídos)", rep.MigrationsCount)
	}

	// matched: plural composto por token + exato
	wantMatched := []string{"Agendamentos", "BloqueiosAgendas", "ListasEsperas", "ProfissionaisEspeciais", "Usuarios"}
	if strings.Join(rep.Matched, ",") != strings.Join(wantMatched, ",") {
		t.Errorf("matched = %v, want %v", rep.Matched, wantMatched)
	}

	// naming pairs: entidade singular ↔ tabela plural (inclui os naive +s)
	wantPairs := []string{"Agendamento→Agendamentos", "BloqueioAgenda→BloqueiosAgendas", "ListaEspera→ListasEsperas", "ProfissionalEspecial→ProfissionaisEspeciais", "Usuario→Usuarios"}
	gotPairs := make([]string, 0, len(rep.NamingPairs))
	for _, p := range rep.NamingPairs {
		gotPairs = append(gotPairs, p.Entity+"→"+p.Table)
	}
	if strings.Join(gotPairs, "|") != strings.Join(wantPairs, "|") {
		t.Errorf("naming pairs = %v, want %v", gotPairs, wantPairs)
	}

	// documentada sem schema (spec à frente)
	if len(rep.DocumentedWithoutSchema) != 1 || rep.DocumentedWithoutSchema[0] != "ValesFakes" {
		t.Errorf("DocumentedWithoutSchema = %v, want [ValesFakes]", rep.DocumentedWithoutSchema)
	}

	// entidade de produto sem documentação
	if len(rep.SchemaWithoutDocs) != 1 || rep.SchemaWithoutDocs[0] != "ProdutoSemDoc" {
		t.Errorf("SchemaWithoutDocs = %v, want [ProdutoSemDoc]", rep.SchemaWithoutDocs)
	}

	// infra Identity separada (AspNet prefix)
	if len(rep.InfraEntities) != 1 || rep.InfraEntities[0] != "IdentityUser" {
		t.Errorf("InfraEntities = %v, want [IdentityUser]", rep.InfraEntities)
	}
}

func TestPluralToken(t *testing.T) {
	cases := map[string]string{
		"especial":  "especiais", // -l → -is
		"usuario":   "usuarios",  // vogal → +s
		"config":    "configs",   // consoante → +s
		"item":      "itens",     // -m → -ns
		"lista":     "listas",    // vogal → +s
		"bloqueio":  "bloqueios", // vogal → +s
		"especiais": "especiais", // já plural
	}
	for in, want := range cases {
		if got := pluralToken(in); got != want {
			t.Errorf("pluralToken(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPluralCompound(t *testing.T) {
	cases := map[string]string{
		"BloqueioAgenda":       "BloqueiosAgendas",
		"ListaEspera":          "ListasEsperas",
		"ProfissionalEspecial": "ProfissionaisEspeciais",
		"Agendamento":          "Agendamentos",
	}
	for in, want := range cases {
		if got := pluralCompound(in); got != want {
			t.Errorf("pluralCompound(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEntitiesMatchLevels(t *testing.T) {
	// nível prefixo: cobre marca com caps internas e plural misto da DM
	cases := []struct{ entity, table string }{
		{"WhatsAppTenantConfig", "WhatsAppTenantConfigs"}, // marca: prefixo direto
		{"BloqueioAgenda", "BloqueiosAgenda"},             // plural misto da própria DM
		{"Agendamento", "Agendamentos"},                   // naive +s
		{"Cliente", "Clientes"},                           // simples
		{"Notificacao", "Notificacoes"},                   // -ção/-ções (normalizeTok)
		{"ReservaListaEspera", "ReservasListaEspera"},     // plural misto por token
	}
	for _, c := range cases {
		if !entitiesMatch(c.entity, c.table) {
			t.Errorf("entitiesMatch(%q, %q) deveria bater (prefixo/plural/token)", c.entity, c.table)
		}
	}
	// negativos reais: não pareia com nomes de outro domínio
	for _, c := range []struct{ entity, table string }{{"Agendamento", "Usuarios"}, {"Cliente", "BloqueiosAgenda"}} {
		if entitiesMatch(c.entity, c.table) {
			t.Errorf("entitiesMatch(%q, %q) NÃO deveria bater", c.entity, c.table)
		}
	}
}
