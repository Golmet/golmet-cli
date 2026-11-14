package cli

import (
	"testing"
)

// parseKey: backspace (0x08) e DEL (0x7F) → keyBack (voltar um passo).
func TestParseKeyBackspaceAndDEL(t *testing.T) {
	if parseKey([]byte{8}) != keyBack {
		t.Errorf("0x08 deveria ser keyBack")
	}
	if parseKey([]byte{127}) != keyBack {
		t.Errorf("0x7F (DEL) deveria ser keyBack")
	}
}

// tecla desconhecida → no-op NUNCA cancel; cancel fica EXCLUSIVO de
// q/Q/esc/ctrl-c (fix do backspace-cancela do operador).
func TestParseKeyUnknownIsNoOpNeverCancel(t *testing.T) {
	for _, b := range []byte{0x99, 0x01, 0x02, 0x05, 0x06, 0x07, 0x11, 0x1F, 'z', '1', '9', '_', '-', 0} {
		if k := parseKey([]byte{b}); k == keyCancel {
			t.Errorf("byte 0x%02X virou cancel (desconhecida deve ser no-op)", b)
		} else if k != "" {
			t.Errorf("byte 0x%02X virou %q (want no-op)", b, k)
		}
	}
	for _, tc := range []struct {
		b    byte
		want string
	}{{'q', keyCancel}, {'Q', keyCancel}, {27, keyCancel}, {3, keyCtrlC}} {
		if got := parseKey([]byte{tc.b}); got != tc.want {
			t.Errorf("parseKey(0x%02X) = %q, want %q", tc.b, got, tc.want)
		}
	}
}

// enter aceita \r e \n (regressão do fix de render).
func TestParseKeyEnterCRAndLF(t *testing.T) {
	if parseKey([]byte{13}) != keyEnter {
		t.Errorf("\\r deveria ser enter (Windows raw)")
	}
	if parseKey([]byte{10}) != keyEnter {
		t.Errorf("\\n deveria ser enter (terminais que entregam LF)")
	}
}
