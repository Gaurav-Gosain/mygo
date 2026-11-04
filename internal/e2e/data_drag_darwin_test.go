//go:build darwin

package e2e

import (
	"github.com/Gaurav-Gosain/mygo"
	"github.com/Gaurav-Gosain/mygo/internal/darwin"
	"github.com/Gaurav-Gosain/mygo/transfer"
)

func dropNativeData(w *mygo.Window, x, y float64, d transfer.Data, ops transfer.Operation) (operation transfer.Operation, dropped, supported bool) {
	mygo.RunOnMain(func() { operation, dropped = darwin.TestDropData(w.NativeHandle(), x, y, d, ops) })
	return operation, dropped, true
}
