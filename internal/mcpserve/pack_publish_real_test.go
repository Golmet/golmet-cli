package mcpserve

// smoke real (M32 — executado na máquina do operador): o preview do
// gm_pack_publish contra o state pack real da org, SEM confirm — o
// comportamento é read-only (nada é publicado). Skip quando o pack não
// existe na máquina (CI).
import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGmPackPublishRealOperatorPackPreview(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("home indisponível")
	}
	packDir := filepath.Join(home, ".config", "opencode", "packs", "techminds")
	if _, serr := os.Stat(filepath.Join(packDir, "pack.yaml")); serr != nil {
		t.Skip("o pack techminds não está nesta máquina")
	}
	cs := session(t, context.Background())
	res, out := callTool(t, cs, "gm_pack_publish", map[string]any{
		"home": home, // SEM confirm — o preview é read-only (o padrão das mutações)
	})
	if !res.IsError {
		t.Fatalf("sem confirm o preview vem como erro (o padrão das mutações): %s", out)
	}
	// o comportamento real do pack da org (estado clean na máquina):
	// o preview resolve o state pack e mostra o estado da subida proposta
	if !strings.Contains(out, "techminds") {
		t.Errorf("o preview deveria resolver o state pack da org:\n%s", out)
	}
}
