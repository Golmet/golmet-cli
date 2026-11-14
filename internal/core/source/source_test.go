package source

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlugStableAndSafe(t *testing.T) {
	a := Slug("https://github.com/Golmet/golmet-core.git")
	b := Slug("https://github.com/Golmet/golmet-core.git")
	if a != b {
		t.Fatalf("slug não determinístico: %q != %q", a, b)
	}
	if a == "" || a == "." || a == ".." {
		t.Fatalf("slug inválido: %q", a)
	}
}

func TestRedactRemovesCredentialsAndSecretQuery(t *testing.T) {
	raw := "https://alice:password@example.com/core.git?token=top-secret&ref=main&api_key=hidden"
	got := Redact(raw)
	for _, secret := range []string{"alice", "password", "top-secret", "hidden"} {
		if strings.Contains(got, secret) {
			t.Fatalf("segredo persistido na origem redigida: %q", got)
		}
	}
	if !strings.Contains(got, "example.com") || !strings.Contains(got, "ref=main") {
		t.Fatalf("origem útil foi perdida na redação: %q", got)
	}
}

func TestLocalGitSourceDirtyFailsClosed(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível")
	}
	repo := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	runGit("init")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("clean"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "README.md")
	runGit("-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(repo, "untracked.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Sync(repo, filepath.Join(t.TempDir(), "cache"))
	if err == nil || !strings.Contains(err.Error(), "fonte Git local suja") {
		t.Fatalf("Git local suja deveria falhar fechado: %v", err)
	}
}

func TestSyncRejectsDirtyGitCache(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível")
	}
	cacheRoot := t.TempDir()
	url := "https://example.com/dirty-cache.git"
	path := filepath.Join(cacheRoot, Slug(url))
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = path
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit("init")
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("clean"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "README.md")
	runGit("-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("dirty"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Sync(url, cacheRoot)
	if err == nil || !strings.Contains(err.Error(), "cache Git sujo") {
		t.Fatalf("cache Git sujo deveria falhar fechado: %v", err)
	}
}
