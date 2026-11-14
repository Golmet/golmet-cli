// Package git isola o spawn do git para o kit (HEAD de repo e resolução
// de objetos por hash — FORMATO §5). Sem git no PATH o adapter degrada:
// Available()==false e o validate emite aviso S-level em vez de erro.
package git

import (
	"os/exec"
	"strings"
)

// Adapter executa git em repos registrados no charter §1.
type Adapter struct {
	available bool
}

// New verifica a disponibilidade do git no PATH.
func New() *Adapter {
	err := exec.Command("git", "--version").Run()
	return &Adapter{available: err == nil}
}

// Available reporta se o git pode ser invocado.
func (a *Adapter) Available() bool { return a.available }

// Head retorna o SHA completo do HEAD do repo em repoPath.
func (a *Adapter) Head(repoPath string) (string, error) {
	out, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// HasObject reporta se o hash existe como objeto no repo (cat-file -e).
func (a *Adapter) HasObject(repoPath, hash string) bool {
	return exec.Command("git", "-C", repoPath, "cat-file", "-e", hash).Run() == nil
}
