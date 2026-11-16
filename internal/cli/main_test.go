package cli

import (
	"os"
	"testing"
)

// TestMain — o home dos testes do cli é SEMPRE um temp dir: o catálogo
// global de organizações (~/.golmet/organizations.json) e qualquer outro
// estado de home NUNCA é tocado nos testes (lição real da M22.1: um teste
// de init --org registrou uma entrada no catálogo do operador).
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "golmet-cli-test-home")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	oldUserHome := userHomeFn
	userHomeFn = func() (string, error) { return tmp, nil }
	code := m.Run()
	userHomeFn = oldUserHome
	os.Exit(code)
}

// isolatedHome — home PRÓPRIO do teste (o catálogo global de organizações
// não vaza entre testes nem para o home real; os fluxos de org que criam/
// registram usam isto).
func isolatedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	old := userHomeFn
	userHomeFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeFn = old })
	return home
}
