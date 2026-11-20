你在 /home/laix/ebook（分支 ui）：MyGo（github.com/egoist/mygo v0.10.0）原生 UI 电子书阅读器，**全平台**（iPhone + 桌面）。引擎**已经做完了**，你只做 UI。

## 现状（先读代码，别重写）
- `internal/engine`：真实的 Rust 引擎 cgo 绑定，`go test ./internal/engine/` 全过。类型见 `internal/engine/types.go`（Book / Chapter / TOCEntry / PageMeta / Bitmap / Location / Selection / Link / Style / Viewport / Engine 接口）。行为约定见 `docs/engine-api.md`。
- `ui/`：上一轮 agent 留下的 **WIP，编译不过**（两代命名混用：`ui/types.go`+`ui/engine.go` 是本地副本，app.go 却在用不带 E 前缀的类型，还有 `minf/maxf/pageForLine` 重复声明）。已经把 WIP 提交在 ui 分支，你可以直接改。

## 你的任务
把 `ui/` 收拾成能跑、能测、能看的样子：

1. **删掉本地副本**：`ui/types.go` 整个删掉；`ui/engine.go` 里凡是重复定义类型/接口的部分删掉，只保留"把 engine 接口适配成 UI 需要的样子"的部分（如果有价值的话）。统一 import `github.com/Laic7092/ebook/internal/engine`。
2. **修完所有编译错误**：`go build ./...` 必须通过。重复的 helper（`minf/maxf/absf/pageForLine` 等）只留一份。
3. **补 main.go**：`mygo.Run` 里挂上 `ui.NewReader(...)`；窗口标题用书名；`internal/engine.New()` 传进去。参考模板现有写法与 `~/go/pkg/mod/github.com/egoist/mygo@v0.10.0/docs/ui/`。
4. **响应式两套布局**（保留并修好 layout.go 的断点设计）：
   - compact（宽 < 720pt，手机）：单页；左/右 1/3 点击翻页、中间切换工具栏；工具栏浮层；目录/设置全屏面板
   - regular（宽 ≥ 720pt，桌面）：左侧目录栏 + 中间页面（≥1100pt 双页）+ 顶部工具栏 + 底部进度；键盘 ←/→/空格/PageUp/PageDown 翻页、Ctrl/⌘ +/- 调字号、鼠标滚轮翻页
   - 共用同一批 widget 构造器，只在 layout.go 里按档位组装
5. **排版时机**：viewport 变化（resize、工具栏显隐改变内边距）防抖 150ms 后重排；重排放后台 job，用 `Window.Update` 回投；重排期间保留旧页，不白屏。光栅化也放后台，带 `pageKey`（页号+scale+墨色+纸色+粗体）缓存，最多留 6 页。
6. **持久化**：上次读到的 spine/页/偏移、字号、行距、主题（纸色/夜间）、加粗，用 `ui.Handle`（读写 + 去抖保存），冷启动恢复到上次位置。
7. **测试**：`ui/` 至少 8 个 `ui.NewTester` 用例（compact 点击翻页、滑动翻页、工具栏显隐、目录跳转、字号变化触发重排、regular 双页、键盘翻页、resize 重排）。测试里用假引擎（本地定义的小接口 + 假实现，或构造 engine 的假数据），不要依赖真实静态库：**注意 `internal/engine` 在 `!cgo` 下会退化成桩**，测试里最好注入自己的 fake。
8. 打开真实书籍：`go tool mygo dev` 时用 `os.Open`/`mygo` 的文档目录 API 打开用户选的 .epub/.txt（桌面用文件对话框，移动端用文档目录里的第一本书）。

## 规则
- `c.Key(key)` 给有状态控件稳定身份；**不要把 `ui.Element` 存进结构体**；慢活不进 UI 线程
- 中文注释，不引入新依赖
- 验收：`go build ./...`、`go test ./...`、`go tool mygo vet .` 全过；`go test ./internal/engine/` 仍然全过
- 完成后在 `ui` 分支提交（不要动 `internal/engine`、`scripts/`、`.github/`）

## 交付
一段说明：改了哪些文件、两套布局结构、测试覆盖、剩余问题。
