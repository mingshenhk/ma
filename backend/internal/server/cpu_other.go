//go:build !windows && !linux

package server

import "errors"

func processCPUSeconds() (float64, error) {
	return 0, errors.New("process CPU metric unavailable on this OS")
}
