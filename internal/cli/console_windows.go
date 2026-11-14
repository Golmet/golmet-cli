//go:build windows

// console_windows.go — compatibilidade de terminal do wizard (Windows):
// o clear ANSI do wizard (\x1b[H\x1b[2J) exige ENABLE_VIRTUAL_TERMINAL_
// PROCESSING no stdout — cmd/conhost legado não tem ligado por default e
// os frames EMPILHAM; e os glifos (braille/◆/✔) exigem codepage UTF-8.
package cli

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVTWin habilita ENABLE_VIRTUAL_TERMINAL_PROCESSING (0x0004) no
// stdout, preservando o modo atual. false = console sem suporte (ou stdout
// não-console) → o dispatch degrada para o modo plain numerado.
func enableVTWin() bool {
	h := windows.Handle(os.Stdout.Fd())
	var old uint32
	if err := windows.GetConsoleMode(h, &old); err != nil {
		return false
	}
	return windows.SetConsoleMode(h, old|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}

// setOutputCodepageUTF8 muda a codepage de output para UTF-8 (65001) e
// devolve a anterior para restauração no exit do wizard (0 = falhou ou
// já era UTF-8 — sem restore; mojibake cosmético melhor que complexidade).
func setOutputCodepageUTF8() uint32 {
	old, err := windows.GetConsoleOutputCP()
	if err != nil || old == 65001 {
		return 0
	}
	if err := windows.SetConsoleOutputCP(65001); err != nil {
		return 0
	}
	return old
}

// restoreOutputCodepage devolve a codepage anterior (cortesia com a sessão
// do operador; 0 = não mudou, no-op; erro ignorado — best-effort).
func restoreOutputCodepage(old uint32) {
	if old != 0 {
		_ = windows.SetConsoleOutputCP(old)
	}
}
