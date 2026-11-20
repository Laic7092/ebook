# ebook

MyGo 原生 UI 的电子书阅读器，排版与渲染复用 [ebook-engine](../ebook-engine)（Rust）：`format → StyledDoc → layout → LayoutResult → render(raster)`。

## 结构

- `main.go`：App 壳。打开书（系统文件对话框 / 「用 eBook 打开」/ 命令行 `-open`）、把书存进 App 自己的书库、记住上次读到哪
- `ui/`：阅读界面（compact / regular 两套布局、翻页、目录、设置、进度）
- `internal/reader`：把 `internal/engine` 接到 `ui.Engine` 的适配层（引擎类型 ↔ UI 类型）
- `internal/engine`：ebook-engine 的 Go 绑定（iOS 上链 Rust 静态库 `ebook-abi`，无 cgo 时返回 `ErrUnsupported`）
- `.github/workflows/ios.yml`：在 macOS runner 上出未签名 `.ipa`（`build/ios-arm64/*-unsigned.ipa`），用 Sideloadly 免费 Apple ID 重签安装

## 本地

引擎是 Rust 静态库，先交叉编译出本机用的那一份（生成到 `internal/engine/native/<os>_<arch>/`，不入库）：

```bash
./scripts/build-engine.sh linux_amd64     # 或 darwin_arm64、darwin_amd64
```

mygo 默认 `CGO_ENABLED=0`（纯 Go 优先），跑真引擎时要显式打开：

```bash
CGO_ENABLED=1 go tool mygo dev            # Linux/Windows 窗口里开发
CGO_ENABLED=1 go run . -open ~/books/demo.epub
go test ./...                             # ui.NewTester，无窗口
```

没有 cgo 时引擎退化成桩：界面照常起来，只是打开书会明确报「引擎不可用」。

## 打开书

- 桌面：界面上的「打开 EPUB / TXT」、命令行 `-open`、文件管理器双击（`mygo.json` 的 `fileAssociations`）
- 手机：系统的文件选择器（`mygo.Dialog.Open`，iOS 上就是 Files 的文档选择器），或用「用 eBook 打开」把书分享进来
- 选中的文件会复制到 App 目录（`<配置目录>/eBook/library/`），因为手机给的是临时副本；下次启动自动接着上次的位置

## IPA

push 到 `main` 或在 Actions 手动运行 `iOS IPA`，产物 `ios-unsigned-ipa`。
