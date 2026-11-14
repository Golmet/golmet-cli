// Package db — serviços de banco de dados do kit (gm-dba determinístico).
// O drift compara TRÊS fontes e reporta divergências — nunca auto-corrige
// (regra 9 do gm-dba). Nesta versão: DATA_MODEL.md ↔ ModelSnapshot EF
// (fontes em arquivo). Schema vivo fica para conexão dedicada.
package db

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// NamingPair é o par entidade (singular, snapshot) ↔ tabela (plural,
// migração/DB) que o DATA_MODEL deve documentar explicitamente.
type NamingPair struct {
	Entity string
	Table  string
}

// Report é o resultado determinístico do drift.
type Report struct {
	Workspace               string
	RepoPath                string
	DocumentedTables        []string // DATA_MODEL.md
	SnapshotEntities        []string // ModelSnapshot (sem generics)
	InfraEntities           []string // Identity/framework (AspNet*, ApplicationUser)
	MigrationsCount         int
	MigrationsPath          string
	Matched                 []string     // documentada ↔ entidade (par confirmado)
	NamingPairs             []NamingPair // entidade singular ↔ tabela plural (documentar)
	DocumentedWithoutSchema []string     // spec à frente (verificar delivery)
	SchemaWithoutDocs       []string     // entidade de produto sem documentação
	LiveChecked             bool         // schema vivo não conectado nesta versão
}

// --- extração: DATA_MODEL.md (tabelas documentadas)

var reTabelaDM = regexp.MustCompile("### Tabela `([A-Za-z0-9_]+)`")

func extractDocumentedTables(dataModelPath string) ([]string, error) {
	data, err := os.ReadFile(dataModelPath)
	if err != nil {
		return nil, fmt.Errorf("DATA_MODEL.md: %w", err)
	}
	var out []string
	seen := map[string]bool{}
	for _, m := range reTabelaDM.FindAllStringSubmatch(string(data), -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out, nil
}

// --- extração: ModelSnapshot EF (entidades = estado final do schema)

var (
	reSnapshotEntity = regexp.MustCompile(`Entity\(\s*"([^"]+)"`)
	reGenericNoise   = regexp.MustCompile(`[<>]`)
)

func extractSnapshotEntities(snapshotPath string) (entities, infra []string, err error) {
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil, nil, fmt.Errorf("ModelSnapshot: %w", err)
	}
	seen := map[string]bool{}
	for _, m := range reSnapshotEntity.FindAllStringSubmatch(string(data), -1) {
		full := m[1]
		name := full[strings.LastIndex(full, ".")+1:]
		if name == "" || reGenericNoise.MatchString(name) {
			continue // ruído de tipo genérico (ex.: "Guid>")
		}
		if strings.HasPrefix(name, "AspNet") || strings.HasPrefix(name, "Identity") || name == "ApplicationUser" {
			if !seen[name] {
				seen[name] = true
				infra = append(infra, name)
			}
			continue
		}
		if !seen[name] {
			seen[name] = true
			entities = append(entities, name)
		}
	}
	sort.Strings(entities)
	sort.Strings(infra)
	return entities, infra, nil
}

// --- descoberta da pasta de migrations (dotnet: Migrations/ com ModelSnapshot)

func findDotnetMigrations(repoPath string) (dir string, count int, err error) {
	var candidates []string
	err = filepath.WalkDir(repoPath, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // sub-árvore ilegível: ignora e segue
		}
		if d.IsDir() && d.Name() == "Migrations" {
			candidates = append(candidates, p)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", 0, fmt.Errorf("varredura do repo: %w", err)
	}
	if len(candidates) == 0 {
		return "", 0, fmt.Errorf("pasta Migrations não encontrada em %s", repoPath)
	}
	if len(candidates) > 1 {
		return "", 0, fmt.Errorf("múltiplas pastas Migrations (%s) — desambigue com --migrations-dir", strings.Join(candidates, "; "))
	}
	dir = candidates[0]
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".cs") && !strings.Contains(name, "Designer") && !strings.Contains(name, "ModelSnapshot") {
			count++
		}
	}
	return dir, count, nil
}

// --- pluralização PT para tokens PascalCase (a regra que o naive +s quebraria)

// pluralToken aplica as regras de plural do PT-BR num token:
// vogal final → +s · -l → -is · -m → -ns · consoante → +s · já plural → intacto.
func pluralToken(tok string) string {
	if tok == "" {
		return tok
	}
	last := tok[len(tok)-1]
	switch last {
	case 's', 'S':
		return tok // já termina em s (plural ou invariável)
	case 'l', 'L':
		return tok[:len(tok)-1] + "is" // animal→animais, especial→especiais
	case 'm', 'M':
		return tok[:len(tok)-1] + "ns" // sistema→sistemas... na verdade -m→-ns (item→itens)
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return tok + "s"
	default:
		return tok + "s" // consoante (config→configs, token→tokens)
	}
}

// splitPascal quebra um identificador PascalCase em tokens
// (ProfissionalDisponibilidade → [profissional, disponibilidade]).
func splitPascal(s string) []string {
	var tokens []string
	var cur strings.Builder
	for _, r := range s {
		if r >= 'A' && r <= 'Z' && cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// pluralCompound pluraliza CADA token PRESERVANDO o PascalCase
// (BloqueioAgenda → BloqueiosAgendas — o naive +s global quebraria).
func pluralCompound(s string) string {
	var out strings.Builder
	for _, tok := range splitPascal(s) {
		out.WriteString(pluralToken(tok))
	}
	return out.String()
}

// entitiesMatch verifica par entidade↔tabela em níveis (do exato ao tolerante):
// 1. igual · 2. plural composto da entidade · 3. PREFIXO em qualquer direção
// (tabela pode estender a entidade ou ser sua raiz). Casos reais de um corpus de validação:
// Agendamento→Agendamentos · BloqueioAgenda→BloqueiosAgenda (plural misto
// na própria DM) · WhatsAppTenantConfig→WhatsAppTenantConfigs (marca com
// caps internas — o splitPascal quebraria, o prefixo resolve).
func entitiesMatch(entity, table string) bool {
	e := strings.ToLower(entity)
	t := strings.ToLower(table)
	if e == t {
		return true
	}
	if strings.HasPrefix(t, e) || strings.HasPrefix(e, t) {
		return true
	}
	ep := strings.ToLower(pluralCompound(entity))
	if ep == t {
		return true
	}
	if strings.HasPrefix(t, ep) || strings.HasPrefix(ep, t) {
		return true
	}
	// token-set singular: compara CONJUNTOS de tokens no singular — cobre
	// plurais mistos (ReservaListaEspera ↔ ReservasListaEspera) e -ção/-ções
	// (Notificação ↔ Notificações) que o prefixo não cobre.
	// PASSA OS ORIGINAIS: o splitPascal precisa das maiúsculas para dividir
	return tokenSingularsMatch(entity, table)
}

// normalizeTok reduz um token ao STEM (singular-normalizado) para
// comparação de conjuntos: -ções/-ção → stem (notificações/notificação →
// notifica) · -coes/-cao → stem (notificacoes/notificacao → notifica) ·
// plural genérico → tira o 's' final (bloqueios → bloqueio).
func normalizeTok(tok string) string {
	l := strings.ToLower(tok)
	switch {
	case strings.HasSuffix(l, "ções"):
		return strings.TrimSuffix(l, "ções")
	case strings.HasSuffix(l, "ção"):
		return strings.TrimSuffix(l, "ção")
	case strings.HasSuffix(l, "coes"):
		return strings.TrimSuffix(l, "coes")
	case strings.HasSuffix(l, "cao"):
		return strings.TrimSuffix(l, "cao")
	}
	return strings.TrimSuffix(l, "s")
}

// tokenSingularsMatch compara os CONJUNTOS de tokens singularizados —
// ordem e posição dos plurais não importam (plurais mistos do PT-BR).
func tokenSingularsMatch(a, b string) bool {
	aToks := map[string]bool{}
	for _, tok := range splitPascal(a) {
		aToks[normalizeTok(tok)] = true
	}
	if len(aToks) == 0 {
		return false
	}
	matched := 0
	for _, tok := range splitPascal(b) {
		if aToks[normalizeTok(tok)] {
			matched++
		}
	}
	return matched == len(aToks)
}

// --- drift

// Drift executa o serviço: DATA_MODEL (workspace/specs) ↔ snapshot +
// migrations (repo). Retorna o Report determinístico — interpretação é
// papel do gm-dba, correção é humana (regra 9).
func Drift(workspace, repoPath string) (*Report, error) {
	dmPath := filepath.Join(workspace, "specs", "DATA_MODEL.md")
	documented, err := extractDocumentedTables(dmPath)
	if err != nil {
		return nil, err
	}
	migDir, migCount, err := findDotnetMigrations(repoPath)
	if err != nil {
		return nil, err
	}
	var snapshotPath string
	entries, _ := os.ReadDir(migDir)
	for _, e := range entries {
		// o SNAPSHOT é o arquivo *_ModelSnapshot.cs — o *.Designer.cs do mesmo
		// nome contém 0 entidades e vem ANTES na ordenação alfabética
		if strings.Contains(e.Name(), "ModelSnapshot") &&
			strings.HasSuffix(e.Name(), ".cs") &&
			!strings.Contains(e.Name(), "Designer") {
			snapshotPath = filepath.Join(migDir, e.Name())
			break
		}
	}
	if snapshotPath == "" {
		return nil, fmt.Errorf("ModelSnapshot não encontrado em %s", migDir)
	}
	entities, infra, err := extractSnapshotEntities(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("snapshot: %w", err)
	}

	rep := &Report{
		Workspace:        workspace,
		RepoPath:         repoPath,
		DocumentedTables: documented,
		SnapshotEntities: entities,
		InfraEntities:    infra,
		MigrationsCount:  migCount,
		MigrationsPath:   migDir,
	}

	docSet := map[string]bool{}
	for _, t := range documented {
		docSet[t] = true
	}
	entSet := map[string]bool{}
	for _, e := range entities {
		entSet[e] = true
	}

	for _, t := range documented {
		matchedPair := ""
		for _, e := range entities {
			if entitiesMatch(e, t) {
				matchedPair = e
				break
			}
		}
		if matchedPair == "" {
			rep.DocumentedWithoutSchema = append(rep.DocumentedWithoutSchema, t)
			continue
		}
		rep.Matched = append(rep.Matched, t)
		if matchedPair != t {
			rep.NamingPairs = append(rep.NamingPairs, NamingPair{Entity: matchedPair, Table: t})
		}
	}
	for _, e := range entities {
		found := false
		for _, t := range documented {
			if entitiesMatch(e, t) {
				found = true
				break
			}
		}
		if !found {
			rep.SchemaWithoutDocs = append(rep.SchemaWithoutDocs, e)
		}
	}
	sort.Strings(rep.Matched)
	sort.Strings(rep.DocumentedWithoutSchema)
	sort.Strings(rep.SchemaWithoutDocs)
	sort.Slice(rep.NamingPairs, func(i, j int) bool { return rep.NamingPairs[i].Entity < rep.NamingPairs[j].Entity })
	return rep, nil
}

// RenderMarkdown gera o relatório DB_DRIFT em markdown (fatos
// determinísticos; a interpretação é papel do gm-dba na conversa).
func (r *Report) RenderMarkdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# DB DRIFT — DATA_MODEL ↔ Migrations\n\n")
	fmt.Fprintf(&b, "> Gerado por `golmet db drift` (determinístico). Interpretação: gm-dba.\n")
	fmt.Fprintf(&b, "> Schema vivo: não conectado nesta rodada.\n\n")
	fmt.Fprintf(&b, "**Fontes:** DATA_MODEL.md (%d tabelas) · Migrations (%d, `%s`) · ModelSnapshot (%d entidades)\n\n",
		len(r.DocumentedTables), r.MigrationsCount, filepath.Base(r.MigrationsPath), len(r.SnapshotEntities))

	fmt.Fprintf(&b, "## 1. Documentado sem backing de schema (%d)\n\n", len(r.DocumentedWithoutSchema))
	if len(r.DocumentedWithoutSchema) == 0 {
		b.WriteString("Nenhum.\n\n")
	} else {
		b.WriteString("| Tabela documentada | Ação sugerida |\n|---|---|\n")
		for _, t := range r.DocumentedWithoutSchema {
			fmt.Fprintf(&b, "| `%s` | Verificar delivery: spec à frente (aguardando implementação) ou atualizar o modelo |\n", t)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "## 2. Schema sem documentação (%d)\n\n", len(r.SchemaWithoutDocs))
	if len(r.SchemaWithoutDocs) == 0 {
		b.WriteString("Nenhum.\n\n")
	} else {
		b.WriteString("| Entidade/Tabela | Ação sugerida |\n|---|---|\n")
		for _, e := range r.SchemaWithoutDocs {
			fmt.Fprintf(&b, "| `%s` | Documentar no DATA_MODEL (ou confirmar como infra) |\n", e)
		}
		b.WriteString("\n")
	}
	if len(r.InfraEntities) > 0 {
		fmt.Fprintf(&b, "Infra de framework (aceitável, citada à parte): %s\n\n", strings.Join(r.InfraEntities, ", "))
	}

	fmt.Fprintf(&b, "## 3. Naming entidade ↔ tabela (%d pares)\n\n", len(r.NamingPairs))
	if len(r.NamingPairs) == 0 {
		b.WriteString("Nenhum — nomes batem.\n\n")
	} else {
		b.WriteString("| Entidade (snapshot) | Tabela (migração/DB) |\n|---|---|\n")
		for _, p := range r.NamingPairs {
			fmt.Fprintf(&b, "| `%s` | `%s` |\n", p.Entity, p.Table)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "## Resumo\n\nmatched: %d · documented-without-schema: %d · schema-without-docs: %d · naming-pairs: %d\n",
		len(r.Matched), len(r.DocumentedWithoutSchema), len(r.SchemaWithoutDocs), len(r.NamingPairs))
	return b.String()
}
