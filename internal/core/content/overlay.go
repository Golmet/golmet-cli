package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type OverlayFile struct {
	Rel  string
	Data []byte
}
type OverlayOptions struct {
	Layer      string
	Source     string
	NoOverride bool
	Force      bool
	// DryRun calcula o plano sem materializar arquivos nem o manifest.
	DryRun bool
}
type OverlayReport struct{ Written, Preserved, Conflicts []string }

// ApplyOverlay aplica arquivos de um pack sobre o materializado. Um arquivo
// gerido por camada anterior pode ser substituído; um arquivo local divergente
// é preservado. no-override transforma colisão em conflito mesmo com conteúdo
// gerido, protegendo o contrato do pack.
func ApplyOverlay(root string, files []OverlayFile, opt OverlayOptions) (*OverlayReport, error) {
	if opt.Layer == "" {
		return nil, fmt.Errorf("layer do overlay vazio")
	}
	prev, err := readContentManifest(ContentManifestPath(root, ""))
	if err != nil {
		return nil, err
	}
	old := previousEntries(prev)
	next := &ContentManifest{Version: "", UpdatedAt: "", Layers: map[string]LayerInfo{}, Files: nil}
	if prev != nil {
		*next = *prev
		if next.Layers == nil {
			next.Layers = map[string]LayerInfo{}
		}
	}
	next.Layers[opt.Layer] = LayerInfo{Source: opt.Source}
	rep := &OverlayReport{}
	for _, f := range files {
		rel := filepath.ToSlash(filepath.Clean(f.Rel))
		if rel == "." || rel == ".." || len(rel) >= 3 && rel[:3] == "../" || filepath.IsAbs(rel) {
			return nil, fmt.Errorf("overlay fora do escopo: %s", f.Rel)
		}
		target := filepath.Join(root, filepath.FromSlash(rel))
		existing, readErr := os.ReadFile(target)
		existingHash := ""
		sameNorm := false // mesmo conteúdo após normalizar EOL — não é colisão real
		if readErr == nil {
			existingHash = sha(existing)
			sameNorm = SameContentNormalized(existing, f.Data)
		}
		oldEntry, hadOld := old[rel]
		conflict := false
		if readErr == nil {
			ownLayer := hadOld && oldEntry.Layer == opt.Layer
			localLayer := hadOld && oldEntry.Layer == "local"
			switch {
			// Precedência local > pack: arquivo local não é sobrescrito
			// sem --force (exceção explícita do FORMAT §2.1) — diferença
			// só de EOL não é customização: cura escrevendo o LF.
			case localLayer && !opt.Force && !sameNorm:
				conflict = true
			case opt.NoOverride && !ownLayer && !sameNorm:
				conflict = true
			case !opt.Force:
				if hadOld && existingHash != oldEntry.SHA256 && !sameNorm {
					conflict = true
				} else if !hadOld && !sameNorm {
					conflict = true
				}
			}
		}
		if conflict {
			rep.Preserved = append(rep.Preserved, rel)
			rep.Conflicts = append(rep.Conflicts, rel)
			continue
		}
		if readErr == nil && existingHash == sha(f.Data) {
			rep.Preserved = append(rep.Preserved, rel)
		} else {
			if !opt.DryRun {
				if err := ensureSafeTarget(root, rel); err != nil {
					return nil, err
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return nil, err
				}
				if err := os.WriteFile(target, f.Data, 0o644); err != nil {
					return nil, err
				}
			}
			rep.Written = append(rep.Written, rel)
		}
		old[rel] = FileEntry{File: rel, SHA256: sha(f.Data), Layer: opt.Layer}
	}
	next.Files = next.Files[:0]
	for _, e := range old {
		next.Files = append(next.Files, e)
	}
	sort.Slice(next.Files, func(i, j int) bool { return next.Files[i].File < next.Files[j].File })
	sort.Strings(rep.Written)
	sort.Strings(rep.Preserved)
	sort.Strings(rep.Conflicts)
	if !opt.DryRun {
		if err := writeContentManifest(ContentManifestPath(root, ""), next); err != nil {
			return nil, err
		}
	}
	return rep, nil
}
