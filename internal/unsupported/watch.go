package unsupported

import "github.com/Gaurav-Gosain/mygo/internal/platform"

func (*Backend) NewFileWatch(func(platform.FileWatchNotice)) (platform.FileWatch, error) {
	return nil, platform.ErrUnsupported
}

func (*Backend) ReplaceFile(target, replacement string) error { return platform.ErrUnsupported }
