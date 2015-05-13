// +build !linux

package taptun

import (
	"fmt"
)

func createInterface(fd uintptr, ifName string, isTAP bool) (string, error) {
	return "", fmt.Errorf("unsupported platform")
}

func setPersistent(fd uintptr, persistent bool) error {
	return fmt.Errorf("unsupported platform")
}

type wrapper struct{}

func wrap(ifce *Interface) (*wrapper, error) {
	return nil, fmt.Errorf("unsupported platform")
}

func (w *wrapper) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("unsupported platform")
}

func (w *wrapper) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("unsupported platform")
}

func (w *wrapper) Stop() bool {
	return false
}
