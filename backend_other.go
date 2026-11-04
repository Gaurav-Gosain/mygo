//go:build !darwin && !(linux && (amd64 || arm64)) && !(windows && (amd64 || arm64)) && !js

package mygo

import (
	"github.com/Gaurav-Gosain/mygo/internal/platform"
	"github.com/Gaurav-Gosain/mygo/internal/unsupported"
)

func newBackend() platform.Backend { return unsupported.New() }
