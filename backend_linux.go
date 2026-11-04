//go:build linux && (amd64 || arm64)

package mygo

import (
	"github.com/Gaurav-Gosain/mygo/internal/linux"
	"github.com/Gaurav-Gosain/mygo/internal/platform"
)

func newBackend() platform.Backend { return linux.New() }

func linkContentRules() { linux.LinkContentRules() }

func linkNowPlaying() { linux.LinkNowPlaying() }

func linkMenuIcons() { linux.LinkMenuIcons() }

func linkContextMenus() { linux.LinkContextMenus() }
