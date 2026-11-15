// L10 — estado INTERNO dos state packs em `<config>/packs/<id>/` (M20): o
// L7 valida o 1º nível da raiz do pack; o L10 vigia o formato interno
// criado pelas skills escritoras — o lint APONTA, o escritor canônico
// corrige (remediação sempre citando a skill responsável):
//
//	A — `rules/*.md` sem gatilho explícito no topo (formato canônico do
//	    gm-rule-keeper: H1 + "> Gatilho: AO …, leia este arquivo ANTES" —
//	    UMA norma por arquivo, lição do monólito de 544 linhas) → warning.
//	B — `accumulation/patterns/<tipo>/` com tipo fora da taxonomia EN
//	    `process|architecture|code|ui|test|data` (constante
//	    `patternTypes` — a mesma lista citada pelo source do
//	    gm-pattern-extractor; era PT até a onda de uniformização) → warning.
//	C — `domains/` sem INDEX.md (índice obrigatório, merge incremental) ou
//	    `domains/<domínio>/` sem KNOWLEDGE.md (formato do gm-domain-research:
//	    fontes primárias URL + data de acesso) → warning.
//	D — working copy com `.git` mas SEM `pack.yaml` → error: a fonte da
//	    verdade do pack sumiu (é o pack.yaml que torna o dir reconhecível
//	    como state pack para o kit inteiro). Critério de working copy =
//	    o mesmo do doctor/engine-config: `.git` presente (dir ou arquivo
//	    de worktree). Dir sem pack.yaml E sem `.git` não é working copy →
//	    fica para os checks existentes, sem duplicar.
package lint

import (
	"os"
	"path/filepath"
	"strings"
)

// ruleTriggerPatterns — padrões de gatilho explícito no topo de uma norma
// (case-insensitive). Formato canônico do gm-rule-keeper: H1 + blockquote
// "> Gatilho: AO <fazer X>, leia este arquivo ANTES de agir" — o marcador
// "gatilho" é o sinal primário (as normas reais variam o verbo: "AO
// configurar…", "AO operar…"); os demais padrões cobrem variantes sem o
// marcador. Constante declarada — ampliar com julgamento, nunca
// silenciosamente.
var ruleTriggerPatterns = []string{
	"gatilho", "ao fazer", "ao criar", "ao operar", "antes de", "leia este", "leia antes",
}

// ruleTriggerHeadLines — janela do topo do arquivo onde o gatilho deve
// aparecer (H1 + blockquote de gatilho + cena de uso).
const ruleTriggerHeadLines = 12

// lintL10PackState varre os working copies de state pack do config e
// emite os findings A-D. Overlay/cache e dir não-reconhecível ficam fora
// (o estado interno é contrato de state pack).
func lintL10PackState(rep *Report, configRoot string) {
	packsDir := filepath.Join(configRoot, "packs")
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		packID := e.Name()
		packDir := filepath.Join(packsDir, packID)
		packRel := "packs/" + packID
		hasGit := dirExists(filepath.Join(packDir, ".git")) || fileExists(filepath.Join(packDir, ".git"))
		if !fileExists(filepath.Join(packDir, "pack.yaml")) {
			if hasGit {
				// D — working copy órfão: o reconhecimento do kit inteiro
				// depende do pack.yaml
				rep.Findings = append(rep.Findings, Finding{
					Check: "L10", Severity: "error", File: packRel,
					Description: "state pack sem pack.yaml — a fonte da verdade do pack sumiu (é o pack.yaml que torna o dir reconhecível como state pack)",
					Fix:         "restaure o pack.yaml (kind: state) ou remova o working copy",
				})
			}
			continue // nem pack nem working copy — checks existentes, sem duplicar
		}
		if !isStatePack(packDir) {
			continue // overlay/cache — o estado interno é contrato de state pack
		}

		// A — normas sem gatilho no topo
		for _, rel := range mdFilesUnder(packDir, "rules") {
			data, rerr := os.ReadFile(filepath.Join(packDir, filepath.FromSlash(rel)))
			if rerr != nil || ruleHasTrigger(string(data)) {
				continue
			}
			rep.Findings = append(rep.Findings, Finding{
				Check: "L10", Severity: "warning", File: packRel + "/" + rel,
				Description: "norma sem gatilho explícito no topo (formato canônico: H1 + \"> Gatilho: AO …\" — uma norma por arquivo)",
				Fix:         "norma sem gatilho — rode o gm-rule-keeper para aplicar o formato canônico",
			})
		}

		// B — patterns fora da taxonomia EN (dono único do check — o L5
		// cuida só de naming)
		patDir := filepath.Join(packDir, "accumulation", "patterns")
		if types, terr := os.ReadDir(patDir); terr == nil {
			for _, t := range types {
				if !t.IsDir() || containsStr(patternTypes, t.Name()) {
					continue
				}
				rep.Findings = append(rep.Findings, Finding{
					Check: "L10", Severity: "warning", File: packRel + "/accumulation/patterns/" + t.Name(),
					Description: "tipo de pattern fora da taxonomia EN (process|architecture|code|ui|test|data) — regressão da taxonomia PT",
					Fix:         "tipo fora da taxonomia EN — renomeie (gm-pattern-extractor define a taxonomia)",
				})
			}
		}

		// C — domains sem contrato
		domDir := filepath.Join(packDir, "domains")
		if doms, derr := os.ReadDir(domDir); derr == nil {
			if !fileExists(filepath.Join(domDir, "INDEX.md")) {
				rep.Findings = append(rep.Findings, Finding{
					Check: "L10", Severity: "warning", File: packRel + "/domains",
					Description: "domains/ sem INDEX.md — o índice é obrigatório (merge incremental)",
					Fix:         "veja gm-domain-research (Split/fundação)",
				})
			}
			for _, d := range doms {
				if !d.IsDir() || fileExists(filepath.Join(domDir, d.Name(), "KNOWLEDGE.md")) {
					continue
				}
				rep.Findings = append(rep.Findings, Finding{
					Check: "L10", Severity: "warning", File: packRel + "/domains/" + d.Name(),
					Description: "domínio sem KNOWLEDGE.md (formato: fontes primárias com URL + data de acesso)",
					Fix:         "veja gm-domain-research (Split/fundação)",
				})
			}
		}
	}
}

// ruleHasTrigger — o topo do arquivo (ruleTriggerHeadLines linhas) contém
// algum padrão de gatilho (case-insensitive)?
func ruleHasTrigger(content string) bool {
	lines := strings.Split(content, "\n")
	if len(lines) > ruleTriggerHeadLines {
		lines = lines[:ruleTriggerHeadLines]
	}
	joined := strings.ToLower(strings.Join(lines, "\n"))
	for _, p := range ruleTriggerPatterns {
		if strings.Contains(joined, p) {
			return true
		}
	}
	return false
}
