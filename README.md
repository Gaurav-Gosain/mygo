> **This is a fork of [egoist/mygo](https://github.com/egoist/mygo)**, kept for
> [tuios](https://github.com/Gaurav-Gosain/tuios)'s desktop app. It is upstream's
> release with these changes on top, each also offered upstream:
>
> - Terminal: scroll the scrollback by pixels on the frame clock (upstream PR: [egoist/mygo#1244](https://github.com/egoist/mygo/pull/1244))
> - Terminal: add rows above the scrollback and keep the view where it is (upstream PR: [egoist/mygo#1246](https://github.com/egoist/mygo/pull/1246))
> - Linux: draw at the monitor's refresh rate on Wayland (upstream PR: [egoist/mygo#1245](https://github.com/egoist/mygo/pull/1245))
> - Linux: hand the P off while GLib waits, so an idle app wakes less (upstream PR: [egoist/mygo#1247](https://github.com/egoist/mygo/pull/1247))
> - Text: lay out a terminal's runs without shaping where each rune keeps its glyph (upstream PR: [egoist/mygo#1248](https://github.com/egoist/mygo/pull/1248))
> - Terminal: read sRGB to linear from a table for minimum contrast (upstream PR: [egoist/mygo#1249](https://github.com/egoist/mygo/pull/1249))
> - Linux: call the GL functions of each batch without reflection (upstream PR: [egoist/mygo#1250](https://github.com/egoist/mygo/pull/1250))
> - Linux: keep the decorations GTK draws while they stay the same (upstream PR: [egoist/mygo#1251](https://github.com/egoist/mygo/pull/1251))
> - Terminal and text fields: stop blinking the caret after the desktop's blink timeout, 10 s by default (branch `patch/terminal-blink-timeout`, not yet offered upstream)
>
> The module path is `github.com/Gaurav-Gosain/mygo`. Tags are
> `v<upstream>-tuios.N`, and the fork follows upstream's releases:
> the `tuios` branch is a release tag, the `patch/*` branches, and one commit
> by `scripts/fork-rename.sh`. MIT, as upstream: see [LICENSE](LICENSE).

<p align="center">
<img width="800" height="360" alt="MyGo!!!!!" src="https://github.com/user-attachments/assets/c3eee9bc-5f22-4032-95b6-776e8381910a" />
</p>

**Desktop apps in Go, with native UI or a web frontend.**

Sponsored by:

<p align="center">
  <a href="https://openuptime.app/?utm_source=github&utm_medium=sponsor&utm_campaign=mygo">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://openuptime.app/sponsor/banner-dark.png">
      <img width="600" alt="OpenUptime: open-source uptime monitoring and status pages" src="https://openuptime.app/sponsor/banner-light.png">
    </picture>
  </a>
</p>

---

Each MyGo window shows one of two kinds of interface, and one app, or one
window, can mix them:

- **A web page**, in the webview the OS already has: WKWebView on macOS,
  WebKitGTK on Linux, WebView2 on Windows. Build the frontend with any web
  tools; it calls your Go code through a TypeScript client generated from it.
- **Native UI**, written in Go alone with package `ui` and
  drawn by MyGo itself on the GPU: no HTML, no JavaScript and no webview, so
  the window opens at once and takes little memory.

Either way, an app is a single Go binary of a few megabytes, focused on low
memory and CPU use.

- **Pure Go, no cgo**: build for every platform from any machine.
- **Typed IPC** for pages: bind Go services, stream values through channels
  and send typed events; the TypeScript client is generated from your Go
  code.
- **A Go UI toolkit** for native UI: flexbox and grid layout, widgets, text
  editing with input methods, virtualized lists, SVG icons, animations,
  screen reader support, and views you test without a window; Liquid Glass
  on every platform with the glass plugin; web pages inside native UI, under
  the menus and dialogs it shows over them.
- **Desktop APIs** for both: windows, menus, tray, dialogs, notifications,
  global shortcuts, deep links, file associations, recent documents, Now
  Playing and the media keys, the system accent color and accessibility
  settings, watching files, and more.
- **Ready to ship**: app bundles and disk images, Windows installers, Debian
  packages, PKGBUILDs for Arch Linux and a Linux install script, code signing, notarization, and signed
  auto-updates with delta updates and an update window in the manner of
  Sparkle.

A window with a web page, whose frontend calls Go:

```go
type Greeter struct{}

// Greet returns a greeting.
func (Greeter) Greet(name string) string { return "Hello, " + name }

func main() {
	mygo.Bind(Greeter{})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Hello", URL: "/"})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

```ts
import { Greeter } from "./mygo"; // generated

document.body.textContent = await Greeter.greet("Ada");
```

A window with native UI, whose view is a Go function of the app's state:

```go
type counter struct{ n int }

func (s *counter) view(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(12).Children(func() {
		ui.Text(c, fmt.Sprint(s.n)).FontSize(40).Bold()
		if ui.PrimaryButton(c, "Increment").Clicked() {
			s.n++
		}
	})
}

func main() {
	s := &counter{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Counter",
			Content: ui.View(s.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

## Getting started

With [Go](https://go.dev/dl/) 1.27.3+ and [Bun](https://bun.sh), create an app
with a web frontend:

```sh
bunx mygo-cli init my-app      # or: npx mygo-cli init my-app
cd my-app
bun run dev
```

An app of native UI needs Go alone:

```sh
go run github.com/Gaurav-Gosain/mygo/cmd/mygo@latest init -template native my-app
cd my-app
go tool mygo dev
```

Read the [documentation](docs/README.md).

## License

MIT
