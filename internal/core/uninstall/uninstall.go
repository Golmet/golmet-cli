// Package uninstall implementa `golmet uninstall` — remoção completa da
// estrutura global do Golmet na máquina, com seleção por camadas. Tudo vem
// SELECIONADO por padrão MENOS o pack (o working copy local é apagado; o
// repositório remoto NUNCA é tocado — remoção manual pelo operador).
// Read-only até a confirmação; nunca toca fora do config root + working
// copies de pack validados como state.
package uninstall

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Golmet/golmet-cli/internal/core/content"
	"github.com/Golmet/golmet-cli/internal/core/pack"
)

// LayerID — ids canônicos das camadas (o que o --only aceita).
const (
	LayerFerramental = "ferramental" // arquivos geridos layer core (manifest)
	LayerEstado      = "estado"      // <config>/.golmet/ inteiro
	LayerPacks       = "packs"       // working copies state packs/<id>/
	LayerPessoal     = "pessoal"     // paths do personal.json (exceto opencode.jsonc)
	LayerNaoGeridos  = "nao-geridos" // config: nem gerido, nem pessoal, nem pack
	LayerEngine      = "engine"      // entradas do Golmet no opencode.jsonc
)

// layerOrder — ordem de EXECUÇÃO da exclusão (engine primeiro; estado antes
// dos packs para o log final informar que o inventário saiu junto).
var layerOrder = []string{LayerEngine, LayerFerramental, LayerPessoal, LayerNaoGeridos, LayerEstado, LayerPacks}

// layerMeta — label/descrição/default para wizard e docs.
type layerMeta struct {
	Label   string
	Hint    string
	Default bool
}

var layerMetaByID = map[string]layerMeta{
	LayerFerramental: {"Ferramental materializado", "agents/, skills/, profile/ (arquivos geridos layer core do manifest)", true},
	LayerEstado:      {"Estado do kit (.golmet/)", "manifest.json + personal.json — o inventário do Golmet nesta máquina", true},
	LayerPacks:       {"Packs instalados (working copy local)", "APENAS o working copy local — o repositório remoto NÃO é tocado (delete manual no GitHub)", false},
	LayerPessoal:     {"Dados pessoais-por-design", "AGENTS.md, profile/client-profile.md, profile/operator-profile.md, cli.json, service.json (opencode.jsonc NÃO — é do engine)", true},
	LayerNaoGeridos:  {"Não-geridos", "arquivos no config que não são geridos, nem pessoais, nem de pack (ex.: plugins/)", true},
	LayerEngine:      {"Registro no engine (opencode.jsonc)", "remove SÓ as entradas do Golmet (MCP golmet, overrides gm-dev-*) — o arquivo e o resto da config permanecem", true},
}

// Options do uninstall. Layers vazio = defaults (tudo menos packs).
type Options struct {
	EngineID string // default opencode
	Home     string // default os.UserHomeDir
	Layers   []string
	DryRun   bool
}

// Layer descoberta: o que seria removido.
type Layer struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Hint    string   `json:"hint"`
	Default bool     `json:"default"`
	Files   []string `json:"files"`          // arquivos (rel ao config)
	Dirs    []string `json:"dirs,omitempty"` // dirs inteiros (rel ao config)
	Count   int      `json:"count"`          // total de alvos (files + dirs)
}

// Plan — descoberta completa (read-only).
type Plan struct {
	EngineID   string  `json:"engine"`
	ConfigRoot string  `json:"config_root"`
	Layers     []Layer `json:"layers"`
}

// LayerResult — resultado da execução por camada.
type LayerResult struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Removed []string `json:"removed"`
	Errors  []string `json:"errors,omitempty"`
	Note    string   `json:"note,omitempty"`
}

// Report do uninstall (CLI Human / JSON).
type Report struct {
	Engine       string        `json:"engine"`
	ConfigRoot   string        `json:"config_root"`
	Layers       []LayerResult `json:"layers"`
	Removed      int           `json:"removed"`
	PackReminder bool          `json:"pack_reminder"`
	EngineNote   string        `json:"engine_note"`
}

// PlanHuman — render textual do plano (dry-run: nada foi apagado).
func (p *Plan) Human() string {
	var b strings.Builder
	fmt.Fprintf(&b, "golmet uninstall — PLANO (dry-run: nada foi apagado) [%s]\n", p.ConfigRoot)
	for _, l := range p.Layers {
		fmt.Fprintf(&b, "\n%s [default: %v] — %d alvo(s)\n", l.Label, l.Default, l.Count)
		fmt.Fprintf(&b, "  %s\n", l.Hint)
		for _, f := range l.Files {
			fmt.Fprintf(&b, "  - %s\n", f)
		}
		for _, d := range l.Dirs {
			fmt.Fprintf(&b, "  - %s/\n", d)
		}
	}
	return b.String()
}

// SelectedDefaults — ids das camadas marcadas por padrão (tudo MENOS packs).
func SelectedDefaults() []string {
	var out []string
	for _, id := range []string{LayerFerramental, LayerEstado, LayerPessoal, LayerNaoGeridos, LayerEngine} {
		if layerMetaByID[id].Default {
			out = append(out, id)
		}
	}
	return out
}

// ValidLayerIDs — ids aceitos pelo --only (ordenados para mensagem estável).
func ValidLayerIDs() []string {
	out := make([]string, 0, len(layerMetaByID))
	for id := range layerMetaByID {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func userHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

func fileExists(p string) bool {
	st, err := os.Lstat(p)
	return err == nil && !st.IsDir()
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func containsStr(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// isStatePack — dir contém pack.yaml com kind: state (mesma semântica do
// doctor/lint — parse via pack.ParseSpec).
func isStatePack(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "pack.yaml"))
	if err != nil {
		return false
	}
	spec, err := pack.ParseSpec(data)
	return err == nil && spec.Kind == pack.KindState
}

// Discover monta o plano a partir do estado REAL da máquina (manifest +
// personal.json + walk do config + pack.yaml). Read-only.
func Discover(opt Options) (*Plan, error) {
	engine := opt.EngineID
	if engine == "" {
		engine = "opencode"
	}
	home := opt.Home
	if home == "" {
		home = userHome()
	}
	if engine != "opencode" {
		return nil, fmt.Errorf("uninstall suporta apenas --engine opencode nesta versão (pedido: %q)", engine)
	}
	if home == "" {
		return nil, fmt.Errorf("sem HOME para o uninstall")
	}
	configRoot := filepath.Join(home, ".config", "opencode")
	if !dirExists(configRoot) {
		return nil, fmt.Errorf("config do engine não existe em %s — nada a desinstalar", configRoot)
	}
	plan := &Plan{EngineID: engine, ConfigRoot: configRoot}

	// fontes de classificação
	manifest, merr := content.ReadContentManifest(configRoot)
	managedCore := []string{}
	managedPack := []string{}
	if merr == nil && manifest != nil {
		for _, e := range manifest.Files {
			if e.File == "" {
				continue
			}
			switch {
			case e.Layer == "core":
				managedCore = append(managedCore, e.File)
			case strings.HasPrefix(e.Layer, "pack:"):
				managedPack = append(managedPack, e.File)
			}
		}
	}
	personal, perr := readPersonal(configRoot)
	if perr != nil {
		personal = nil
	}

	packDirs := discoverPackDirs(configRoot)
	packSet := map[string]bool{}
	for _, d := range packDirs {
		packSet[d] = true
	}

	usedInOtherLayer := func(rel string) bool {
		if rel == "opencode.jsonc" || rel == ".golmet/manifest.json" || rel == ".golmet/personal.json" {
			return true
		}
		if containsStr(managedCore, rel) || containsStr(managedPack, rel) {
			return true
		}
		if personal != nil && personal.Match(rel) {
			return true
		}
		for _, d := range packDirs {
			if strings.HasPrefix(rel, d+"/") || rel == d {
				return true
			}
		}
		return false
	}

	unmanaged := []string{}
	root, _ := filepath.Abs(configRoot)
	walkErr := filepath.WalkDir(configRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			n := d.Name()
			if n == ".git" || n == ".golmet" {
				return filepath.SkipDir
			}
			if rel, rerr := filepath.Rel(root, p); rerr == nil && (rel == "packs" || isStatePack(p)) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if usedInOtherLayer(rel) {
			return nil
		}
		if personal != nil && personal.Match(rel) {
			return nil
		}
		unmanaged = append(unmanaged, rel)
		return nil
	})
	_ = walkErr
	sort.Strings(unmanaged)

	// --- camadas na ordem de EXIBIÇÃO (1..6) ---
	add := func(id string, files, dirs []string) {
		meta := layerMetaByID[id]
		sort.Strings(files)
		sort.Strings(dirs)
		plan.Layers = append(plan.Layers, Layer{
			ID: id, Label: meta.Label, Hint: meta.Hint, Default: meta.Default,
			Files: files, Dirs: dirs, Count: len(files) + len(dirs),
		})
	}

	// 1. ferramental — geridos layer core (AGENTS.template.md nunca é
	// depositado — M12; nonDepositTemplates não chega ao manifest)
	add(LayerFerramental, managedCore, nil)
	// 2. estado do kit
	add(LayerEstado, nil, []string{".golmet"})
	// 3. packs — working copies state (DESELECIONADO por padrão)
	add(LayerPacks, managedPack, packDirs)
	// 4. pessoal — paths do personal.json, exceto opencode.jsonc (do engine)
	var pessoal []string
	if personal != nil {
		for _, p := range personal.All() {
			if p != "opencode.jsonc" && fileExists(filepath.Join(configRoot, filepath.FromSlash(p))) {
				pessoal = append(pessoal, p)
			}
		}
	}
	add(LayerPessoal, pessoal, nil)
	// 5. não-geridos
	add(LayerNaoGeridos, unmanaged, nil)
	// 6. engine — entradas do Golmet no opencode.jsonc
	engineTargets := detectEngineEntries(configRoot)
	meta := layerMetaByID[LayerEngine]
	plan.Layers = append(plan.Layers, Layer{
		ID: LayerEngine, Label: meta.Label, Hint: meta.Hint, Default: meta.Default,
		Files: engineTargets, Count: len(engineTargets),
	})

	// ordena o plano na ordem de execução para o relatório consistente
	sortPlanByExecution(plan)
	return plan, nil
}

func sortPlanByExecution(plan *Plan) {
	rank := map[string]int{}
	for i, id := range layerOrder {
		rank[id] = i
	}
	ls := plan.Layers
	sort.SliceStable(ls, func(i, j int) bool { return rank[ls[i].ID] < rank[ls[j].ID] })
}

func discoverPackDirs(configRoot string) []string {
	packsDir := filepath.Join(configRoot, "packs")
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && isStatePack(filepath.Join(packsDir, e.Name())) {
			out = append(out, "packs/"+e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// readPersonal — parse do personal.json (mesmo schema do doctor).
func readPersonal(configRoot string) (*personalPaths, error) {
	data, err := os.ReadFile(filepath.Join(configRoot, ".golmet", "personal.json"))
	if err != nil {
		return nil, err
	}
	var pf struct {
		Version int      `json:"version"`
		Paths   []string `json:"paths"`
	}
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	return &personalPaths{paths: pf.Paths}, nil
}

type personalPaths struct{ paths []string }

// Match — o path (slash) está designado pessoal-por-design?
func (p *personalPaths) Match(rel string) bool {
	if p == nil {
		return false
	}
	for _, x := range p.paths {
		if x == rel {
			return true
		}
	}
	return false
}

// All — todos os paths designados (para a camada 4).
func (p *personalPaths) All() []string {
	if p == nil {
		return nil
	}
	return p.paths
}

// detectEngineEntries — entradas do Golmet detectadas no opencode.jsonc
// (contagem para a camada 6; detecção textual tolerante a jsonc).
func detectEngineEntries(configRoot string) []string {
	data, err := os.ReadFile(filepath.Join(configRoot, "opencode.jsonc"))
	if err != nil {
		return nil
	}
	s := string(data)
	var out []string
	if strings.Contains(s, "\"golmet\"") {
		out = append(out, "opencode.jsonc: mcp.golmet")
	}
	if strings.Contains(s, "\"gm-dev-backend\"") {
		out = append(out, "opencode.jsonc: agent.gm-dev-backend (permission)")
	}
	if strings.Contains(s, "\"gm-dev-frontend\"") {
		out = append(out, "opencode.jsonc: agent.gm-dev-frontend (permission)")
	}
	return out
}

// Report e execução ----

func (r *Report) Human() string {
	var b strings.Builder
	fmt.Fprintf(&b, "golmet uninstall — %s [%s]\n", r.ConfigRoot, r.Engine)
	for _, l := range r.Layers {
		fmt.Fprintf(&b, "\n%s\n", l.Label)
		if len(l.Removed) == 0 && len(l.Errors) == 0 {
			fmt.Fprintf(&b, "  (nada — não selecionada ou sem alvos)\n")
			continue
		}
		for _, f := range l.Removed {
			fmt.Fprintf(&b, "  removido: %s\n", f)
		}
		for _, e := range l.Errors {
			fmt.Fprintf(&b, "  erro: %s\n", e)
		}
		if l.Note != "" {
			fmt.Fprintf(&b, "  nota: %s\n", l.Note)
		}
	}
	if r.PackReminder {
		fmt.Fprintf(&b, "\n⚠️ PACKS: apenas o working copy LOCAL foi apagado. O repositório remoto\n")
		fmt.Fprintf(&b, "   (ex.: github.com/<org>/<pack-repo>) NÃO é tocado — delete-o\n")
		fmt.Fprintf(&b, "   manualmente no GitHub se quiser remover de vez.\n")
	}
	fmt.Fprintf(&b, "\n%d arquivo(s)/dir(s) removido(s).\n", r.Removed)
	fmt.Fprintf(&b, "%s\n", r.EngineNote)
	return b.String()
}

// Run executa a exclusão das camadas selecionadas, na ordem de execução
// (engine → ferramental → pessoal → não-geridos → estado → packs).
func Run(plan *Plan, selected []string) (*Report, error) {
	rep := &Report{Engine: plan.EngineID, ConfigRoot: plan.ConfigRoot, Layers: []LayerResult{}}
	byID := map[string]Layer{}
	for _, l := range plan.Layers {
		byID[l.ID] = l
	}
	wasSelected := func(id string) bool { return containsStr(selected, id) }

	removeFile := func(lr *LayerResult, configRoot, rel string) {
		p := filepath.Join(configRoot, filepath.FromSlash(rel))
		if !fileExists(p) {
			return // idempotente — ausente é no-op
		}
		if err := os.Remove(p); err != nil {
			lr.Errors = append(lr.Errors, rel+": "+err.Error())
			return
		}
		lr.Removed = append(lr.Removed, rel)
	}
	removeDir := func(lr *LayerResult, configRoot, rel string) {
		p := filepath.Join(configRoot, filepath.FromSlash(rel))
		if !dirExists(p) {
			return
		}
		// segurança: não seguir symlink (RemoveAll em symlink remove o link;
		// aqui exigimos dir real antes de RemoveAll)
		if st, err := os.Lstat(p); err != nil || !st.IsDir() {
			lr.Errors = append(lr.Errors, rel+": não é um diretório regular")
			return
		}
		if err := os.RemoveAll(p); err != nil {
			lr.Errors = append(lr.Errors, rel+": "+err.Error())
			return
		}
		lr.Removed = append(lr.Removed, rel+"/")
	}

	runLayer := func(id string) {
		l, ok := byID[id]
		if !ok || !wasSelected(id) {
			return
		}
		lr := LayerResult{ID: id, Label: l.Label, Removed: []string{}}
		switch id {
		case LayerEngine:
			removed, errs, note := uninstallEngineEntries(plan.ConfigRoot)
			lr.Removed = removed
			lr.Errors = errs
			lr.Note = note
		case LayerEstado:
			removeDir(&lr, plan.ConfigRoot, ".golmet")
			lr.Note = "inventário do Golmet removido desta máquina"
		case LayerPacks:
			for _, d := range l.Dirs {
				removeDir(&lr, plan.ConfigRoot, d)
			}
			for _, f := range l.Files {
				removeFile(&lr, plan.ConfigRoot, f)
			}
			rep.PackReminder = true
			lr.Note = "apenas o working copy LOCAL - o repositório remoto NÃO é tocado (delete manual no GitHub)"
		default:
			for _, f := range l.Files {
				removeFile(&lr, plan.ConfigRoot, f)
			}
		}
		rep.Layers = append(rep.Layers, lr)
	}
	for _, id := range layerOrder {
		runLayer(id)
	}
	// o pack working copy NUNCA vai ao remote: removido ou mantido, o operador
	// merece o aviso (remote só sai por delete manual no GitHub)
	if _, hasPacks := byID["packs"]; hasPacks {
		rep.PackReminder = true
	}

	// dirs vazios que o kit criou (nunca opencode.jsonc; nunca dir com conteúdo)
	rep.Layers = append(rep.Layers, removeEmptyKitDirs(plan.ConfigRoot, selected, byID))

	for _, l := range rep.Layers {
		rep.Removed += len(l.Removed)
	}
	rep.EngineNote = "o engine opencode continua instalado — o Golmet apenas saiu dele"
	return rep, nil
}

// removeEmptyKitDirs — após a exclusão, remove dirs de 1º nível criados pelo
// kit que ficarem VAZIOS. Nunca toca opencode.jsonc nem dirs com conteúdo.
func removeEmptyKitDirs(configRoot string, selected []string, byID map[string]Layer) LayerResult {
	lr := LayerResult{ID: "vazios", Label: "Diretórios vazios removidos", Removed: []string{}}
	kitDirs := []string{"agents", "skills", "profile", "packs", "plugins", "knowledge"}
	// prune bottom-up: remove sub-dirs vazios e depois o próprio dir se ficar vazio
	var prune func(dir string) (removedSelf bool, stillExists bool)
	prune = func(dir string) (removedSelf bool, stillExists bool) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return false, false // inexistente - no-op silencioso
		}
		for _, e := range entries {
			if e.IsDir() {
				sub := filepath.Join(dir, e.Name())
				if removed, _ := prune(sub); removed {
					lr.Removed = append(lr.Removed, sub+string(os.PathSeparator))
				}
			}
		}
		entries, err = os.ReadDir(dir)
		if err != nil {
			return false, true
		}
		if len(entries) == 0 {
			if err := os.Remove(dir); err == nil {
				lr.Removed = append(lr.Removed, dir+string(os.PathSeparator))
				return true, false
			}
		}
		return false, true
	}
	for _, d := range kitDirs {
		p := filepath.Join(configRoot, d)
		if dirExists(p) {
			prune(p)
		}
	}
	return lr
}

// ---- camada 6 — edição cirúrgica do opencode.jsonc ----

// uninstallEngineEntries remove as entradas do Golmet do opencode.jsonc
// (mcp.golmet, agent.gm-dev-backend/frontend) — o arquivo e o resto da
// config permanecem. Idempotente. Se o arquivo tiver comentários (jsonc),
// a remoção é por bloco (brace-matching consciente de strings/comentários)
// e o resultado é VALIDADO (strip de comentários + json.Valid) antes de
// gravar — inválido = arquivo intacto + erro.
func uninstallEngineEntries(configRoot string) (removed, errs []string, note string) {
	path := filepath.Join(configRoot, "opencode.jsonc")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, "opencode.jsonc não existe — nada a remover"
		}
		return nil, []string{"opencode.jsonc: " + err.Error()}, ""
	}
	orig := string(data)
	changed := false
	for _, key := range []string{"golmet", "gm-dev-backend", "gm-dev-frontend"} {
		next, removedNow := removeJSONCKey([]byte(orig), key)
		if removedNow {
			orig = string(next)
			removed = append(removed, "opencode.jsonc: entrada \""+key+"\" removida")
			changed = true
		}
	}
	if !changed {
		return nil, nil, "nenhuma entrada do Golmet no opencode.jsonc - no-op"
	}
	// normaliza vírgulas pendentes antes de }/] - remover a ÚLTIMA key de um
	// objeto deixa a vírgula da key anterior pendente (JSONC tolera; o JSON
	// resultante precisa ser válido)
	orig = stripDanglingCommas(orig)
	// validação: o resultado (jsonc → json puro: sem comentários, sem
	// vírgulas trailing — jsonc permite, json puro não; best-effort
	// determinístico) precisa ser JSON válido antes de gravar
	if !json.Valid(stripJSONCToJSON([]byte(orig))) {
		return nil, []string{"opencode.jsonc: a remoção produziria JSON inválido — arquivo INTACTO, remova as chaves manualmente (mcp.golmet, agent.gm-dev-*)"}, ""
	}
	if err := os.WriteFile(path, []byte(orig), 0o644); err != nil {
		return nil, []string{"opencode.jsonc: " + err.Error()}, ""
	}
	note = "o arquivo opencode.jsonc e o resto da config do engine permanecem"
	return removed, errs, note
}

// removeJSONCKey remove a entrada `"key": { ... }` do jsonc (objeto flat ou
// aninhado), com brace-matching consciente de strings e comentários. Ajusta
// vírgula trailing/leading para manter o JSON válido. Idempotente: sem a
// chave → (data, false).
func removeJSONCKey(data []byte, key string) ([]byte, bool) {
	s := string(data)
	needle := "\"" + key + "\""
	start := -1
	for attempt := 0; ; attempt++ {
		i := indexOfKey(s, needle, attempt)
		if i < 0 {
			return data, false
		}
		// do key: pula espaços/comentários até ':' e depois até '{'
		j := skipWSComments(s, i+len(needle))
		if j >= len(s) || s[j] != ':' {
			continue
		}
		j = skipWSComments(s, j+1)
		if j >= len(s) || s[j] != '{' {
			continue
		}
		end, ok := matchBrace(s, j)
		if !ok {
			continue
		}
		start = i
		// consome vírgula trailing
		k := skipWSComments(s, end+1)
		if k < len(s) && s[k] == ',' {
			end = k
		} else {
			// vírgula leading: se o char anterior (pulando ws) é ',', remove-a
			p := skipWSCommentsBack(s, start-1)
			if p >= 0 && s[p] == ',' {
				start = p
			}
		}
		out := s[:start] + s[end+1:]
		return []byte(out), true
	}
}

// indexOfKey acha a N-ésima ocorrência de needle FORA de strings/comentários.
func indexOfKey(s, needle string, n int) int {
	inStr := false
	escaped := false
	for x := 0; x < len(s); x++ {
		ch := s[x]
		if inStr {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inStr = false
			}
			continue
		}
		switch ch {
		case '"':
			if strings.HasPrefix(s[x:], needle) {
				if n == 0 {
					return x
				}
				n--
				x += len(needle) - 1
				continue
			}
			inStr = true
		case '/':
			if x+1 < len(s) && s[x+1] == '/' {
				for x < len(s) && s[x] != '\n' {
					x++
				}
			} else if x+1 < len(s) && s[x+1] == '*' {
				x += 2
				for x+1 < len(s) && !(s[x] == '*' && s[x+1] == '/') {
					x++
				}
				x++
			}
		}
	}
	return -1
}

// skipWSComments pula espaços e comentários a partir de i.
func skipWSComments(s string, i int) int {
	for i < len(s) {
		switch {
		case s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r':
			i++
		case s[i] == '/' && i+1 < len(s) && s[i+1] == '/':
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case s[i] == '/' && i+1 < len(s) && s[i+1] == '*':
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i++
		default:
			return i
		}
	}
	return i
}

// skipWSCommentsBack pula espaços/comentários para trás a partir de i-1.
func skipWSCommentsBack(s string, i int) int {
	for i >= 0 {
		switch {
		case s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r':
			i--
		case i-1 >= 0 && s[i] == '/' && s[i-1] == '*':
			i -= 2
			for i-1 >= 0 && !(s[i] == '*' && s[i+1] == '/') {
				i--
			}
			i--
		default:
			return i
		}
	}
	return -1
}

// matchBrace — dado o índice do '{', devolve o índice do '}' correspondente,
// ignorando strings e comentários.
func matchBrace(s string, open int) (int, bool) {
	depth := 0
	inStr := false
	escaped := false
	for x := open; x < len(s); x++ {
		ch := s[x]
		if inStr {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inStr = false
			}
			continue
		}
		switch ch {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return x, true
			}
		case '/':
			if x+1 < len(s) && s[x+1] == '/' {
				for x < len(s) && s[x] != '\n' {
					x++
				}
			} else if x+1 < len(s) && s[x+1] == '*' {
				x += 2
				for x+1 < len(s) && !(s[x] == '*' && s[x+1] == '/') {
					x++
				}
				x++
			}
		}
	}
	return -1, false
}

// stripJSONCComments remove comentários (// e /* */) respeitando strings.
func stripJSONCComments(data []byte) []byte {
	s := string(data)
	var b strings.Builder
	inStr := false
	escaped := false
	for x := 0; x < len(s); x++ {
		ch := s[x]
		if inStr {
			b.WriteByte(ch)
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inStr = false
			}
			continue
		}
		switch {
		case ch == '"':
			inStr = true
			b.WriteByte(ch)
		case ch == '/' && x+1 < len(s) && s[x+1] == '/':
			for x < len(s) && s[x] != '\n' {
				x++
			}
			if x < len(s) {
				b.WriteByte('\n')
			}
		case ch == '/' && x+1 < len(s) && s[x+1] == '*':
			x += 2
			for x+1 < len(s) && !(s[x] == '*' && s[x+1] == '/') {
				x++
			}
			x++
		default:
			b.WriteByte(ch)
		}
	}
	return []byte(b.String())
}

// stripJSONCToJSON — jsonc → json puro (validação): remove comentários E
// vírgulas trailing (jsonc permite `, }`; json puro não). Loop até ponto
// fixo — vírgulas em cascata (`, , }`) colapsam juntas. Best-effort
// determinístico para a VALIDAÇÃO da camada engine (falso-positivo aborta
// sem gravar — seguro por design).
func stripJSONCToJSON(data []byte) []byte {
	s := stripJSONCComments(data)
	re := regexp.MustCompile(`,(\s*[}\]])`)
	for {
		next := re.ReplaceAll(s, []byte("$1"))
		if string(next) == string(s) {
			return s
		}
		s = next
	}
}
