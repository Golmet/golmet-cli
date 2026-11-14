//go:build !windows

// console_other.go — terminais POSIX já processam ANSI/VT e rodam UTF-8:
// wrappers no-op para os mesmos pontos de chamada do dispatch.
package cli

func enableVTWin() bool { return true }

func setOutputCodepageUTF8() uint32 { return 0 } // nada a restaurar

func restoreOutputCodepage(uint32) {}
