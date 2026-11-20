# ebook

MyGo 原生 UI 的电子书阅读器，排版与渲染复用 [ebook-engine](../ebook-engine)（Rust）：`format → StyledDoc → layout → LayoutResult → render(raster)`。

## 结构

- `main.go`：native UI 外壳（窗口、阅读界面）
- `internal/engine`：ebook-engine 的 Go 绑定（iOS 上链 Rust 静态库 `ebook-abi`，无 cgo 时返回 `ErrUnsupported`）
- `.github/workflows/ios.yml`：在 macOS runner 上出未签名 `.ipa`（`build/ios-arm64/*-unsigned.ipa`），用 Sideloadly 免费 Apple ID 重签安装

## 本地

```bash
go tool mygo dev          # Linux 窗口里开发
go test ./...             # ui.NewTester，无窗口
CGO_ENABLED=1 go test ./...   # 引擎的 cgo 绑定与 golden
```

## IPA

push 到 `main` 或在 Actions 手动运行 `iOS IPA`，产物 `ios-unsigned-ipa`。
