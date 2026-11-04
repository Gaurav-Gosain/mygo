//go:build js

package mygo

import (
	"github.com/Gaurav-Gosain/mygo/internal/platform"
	"github.com/Gaurav-Gosain/mygo/internal/web"
)

func newBackend() platform.Backend { return web.New() }
