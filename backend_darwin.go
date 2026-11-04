//go:build darwin

package mygo

import (
	"github.com/Gaurav-Gosain/mygo/internal/darwin"
	"github.com/Gaurav-Gosain/mygo/internal/platform"
)

func newBackend() platform.Backend { return darwin.New() }

func enableNotificationSenders() { darwin.EnableNotificationSenders() }

func linkContentRules() { darwin.LinkContentRules() }

func linkNowPlaying() { darwin.LinkNowPlaying() }

func linkMenuIcons() { darwin.LinkMenuIcons() }

func linkContextMenus() { darwin.LinkContextMenus() }
