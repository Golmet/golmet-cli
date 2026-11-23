// pack_mcp_test.go — camada CLI do `pack mcp status/sync`: roteamento,
// mensagens de vazio e saída JSON. Só cenários que NÃO executam CLI de
// engine nem node (os fluxos de registro com injeção estão nos testes do
// core — internal/core/install/mcp_pack_test.go).
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedStatePackCLI cria working copy de state pack no root do opencode da
// home informada (SEM mcp/server.js — status lista como server-ausente e
// sync não executa nada; sem sondagem de engine no teste).
func seedStatePackCLI(t *testing.T, home, id string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "opencode", "packs", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := "id: " + id + "\nversion: 1.0.0\nname: Pack " + id + "\nkind: state\n"
	if err := os.WriteFile(filepath.Join(dir, "pack.yaml"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPackMCPStatusNoStatePacks(t *testing.T) {
	out, err := runCmd(t, "pack", "mcp", "status", "--home", t.TempDir())
	if err != nil {
		t.Fatalf("pack mcp status: %v\n%s", err, out)
	}
	if !strings.Contains(out, "nenhum state pack sincronizado") {
		t.Errorf("mensagem de vazio ausente: %s", out)
	}
}

func TestPackMCPStatusServerAusente(t *testing.T) {
	home := t.TempDir()
	seedStatePackCLI(t, home, "techminds")
	out, err := runCmd(t, "pack", "mcp", "status", "--home", home)
	if err != nil {
		t.Fatalf("pack mcp status: %v\n%s", err, out)
	}
	for _, want := range []string{"techminds", "opencode", "pack-techminds", "server-ausente"} {
		if !strings.Contains(out, want) {
			t.Errorf("saída sem %q: %s", want, out)
		}
	}
}

func TestPackMCPStatusJSON(t *testing.T) {
	home := t.TempDir()
	seedStatePackCLI(t, home, "techminds")
	out, err := runCmd(t, "pack", "mcp", "status", "--home", home, "--json")
	if err != nil {
		t.Fatalf("pack mcp status --json: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"state"`) || !strings.Contains(out, `"server-ausente"`) {
		t.Errorf("json sem campos esperados: %s", out)
	}
}

func TestPackMCPSyncNoEligible(t *testing.T) {
	home := t.TempDir()
	seedStatePackCLI(t, home, "semserver") // state pack SEM mcp/server.js
	out, err := runCmd(t, "pack", "mcp", "sync", "--home", home)
	if err != nil {
		t.Fatalf("pack mcp sync sem elegível deveria ser suave: %v\n%s", err, out)
	}
	if !strings.Contains(out, "nenhum state pack com mcp/server.js encontrado") {
		t.Errorf("mensagem de vazio ausente: %s", out)
	}
}

func TestPackMCPSyncNoStatePacks(t *testing.T) {
	out, err := runCmd(t, "pack", "mcp", "sync", "--home", t.TempDir())
	if err != nil {
		t.Fatalf("pack mcp sync: %v\n%s", err, out)
	}
	if !strings.Contains(out, "nenhum state pack com mcp/server.js encontrado") {
		t.Errorf("mensagem de vazio ausente: %s", out)
	}
}

// sync com FALHA de registro → exit 1 (parcial): PATH vazio derruba a CLI
// do engine (lookPath determinístico sem injeção) → outcome manual, os
// resultados são impressos e o comando sai 1 (precedente Partial do install).
func TestPackMCPSyncPartialExitsOne(t *testing.T) {
	home := t.TempDir()
	seedStatePackCLIWithServer(t, home, "techminds")
	t.Setenv("PATH", "")
	out, err := runCmd(t, "pack", "mcp", "sync", "--home", home)
	if err == nil {
		t.Fatalf("sync com falha deveria sair 1, saiu 0:\n%s", out)
	}
	if code, ok := ExitCodeOf(err); !ok || code != 1 {
		t.Fatalf("exit code = %v (ok=%v), want 1", code, ok)
	}
	if !strings.Contains(out, "manual") {
		t.Errorf("outcome manual esperado na saída: %s", out)
	}
}

// seedStatePackCLIWithServer — working copy de state pack COM mcp/server.js
// (elegível ao sync). Sem sondagem de engine no seed — a execução externa
// do teste é controlada pelo PATH.
func seedStatePackCLIWithServer(t *testing.T, home, id string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "opencode", "packs", id, "mcp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := "id: " + id + "\nversion: 1.0.0\nname: Pack " + id + "\nkind: state\n"
	if err := os.WriteFile(filepath.Join(dir, "..", "pack.yaml"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "server.js"), []byte("// server\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
