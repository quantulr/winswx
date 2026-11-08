# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概览

winswx 是一个原生 Windows 服务管理器：Go + [MyGo](https://github.com/egoist/mygo) 原生 UI，WinSW 作为服务 wrapper。用户数据目录 `services` 下的每个子目录是一个服务，存放该服务的 WinSW XML 配置。界面支持英文和简体中文，并从 GitHub 上的签名 Release 自动更新。

Go module 名是 `mygo`（不是 `github.com/quantulr/winswx`），入口在 `cmd/winswx`。`mygo.json` 是构建、版本和更新配置的唯一来源。

## 常用命令

```sh
# 构建或测试前必须下载随包携带的 WinSW（已被 .gitignore 忽略）
go run ./scripts/fetch-winsw      # 或 go generate ./...

go tool mygo dev                  # 开发运行，热重载
go tool mygo build                # 构建当前平台
go tool mygo build -platform windows/amd64
go tool mygo vet .                # go vet + MyGo UI 生命周期检查，改动 UI 后运行
go vet ./...

go test ./...
go test ./internal/app -run TestMoreMenuAnimates   # 运行单个测试
go test -race ./internal/app

# 修改 internal/locales/*.json 后重新生成访问器
go tool mygo generate
go tool mygo i18n check
```

测试是无窗口的 MyGo UI 测试（`ui.NewTester`），依赖 `resources/bin/winsw.exe` 存在；`internal/app/testmain_test.go` 已把 `PathResources` 指向 `../../resources`。

## 架构

**启动与生命周期** — `cmd/winswx/main.go` 先请求单实例锁：第二次启动会通过 `App.OnSecondInstance` → `Focus()` 把已有窗口带到前台。随后注册 `native` updater 插件，在 `WhenReady` 时调用 `OpenWindow`。窗口 640×480，隐藏原生标题栏，`StateKey: "main"` 记住位置，内容为 `ui.View(a.view)`；窗口获得焦点时重新 `load()`。

**状态与线程** — `internal/app` 中的 `App` 保存所有模型状态，`view` 在主线程根据它构建界面。WinSW 查询约需 0.4 秒：`refreshStatuses()` 在后台以最多 4 个并发查询，用 `queryGen` 丢弃过期结果，并通过 `win.Update` 回写；`run()` 异步执行服务命令并维护 `busy` map。不要在 UI 线程做阻塞 I/O，也不要把 `ui.Element` 或 `*ui.Context` 存进 `App`（只存应用状态、`ui.Handle` 等持久数据）。

**服务文件** — `load()` 创建并读取 `%AppData%\winswx\services`，保留已知状态，并给有配置的服务标上 `statusChecking`。`serviceConfig()` 的解析规则：优先 `<目录>/<名称>.xml`，否则取目录中唯一的 `.xml`；存在多个 XML 时视为无配置。新增服务写 `<id>`、`<executable>`、`<arguments>`；编辑用 `updateServiceXML()` 做 token 级重写，只替换这三个顶层元素，保留其余未知配置。

**WinSW 调用** — 服务状态必须解析 `winsw status <xml>` 的 **stdout**（systemd 风格，如 `Active (running)`）。WinSW 3.0 对运行中的服务退出码为 1，因此不能靠退出码判断；仅在 stdout 无法解析且退出码为 1060 时判为未安装。

**运行中的服务与更新** — 所有 install/start/stop/restart/uninstall 命令都通过 `serviceWinsw()` 从 `%AppData%\winswx\bin\winsw.exe` 执行，该文件在首次使用时从 `resources/bin/winsw.exe` 复制。这样运行中的服务不会锁住 app 安装目录，更新或卸载才能替换 app 文件；不要改回直接调用 bundled 路径。

**界面结构** — `view.go` 负责主界面：花瓣渐变背景、可拖动的标题条、固定 200×112 的卡片网格（窗口缩放时列数变化、卡片宽度不变）、glass 材质。卡片按钮按状态显示 Start/Stop/Install，其余操作在 More 菜单中。More 菜单和对话框都使用手写的开/关动画：`moreOpen`/`moreTarget`/`moreFresh` 与 `dlgOpen`/`dlgTarget`/`dlgFresh`。关闭时先把 target 设为 0，保留元素继续绘制，直到动画进度小于 0.01 才从树中移除；点击外部或按 Escape 时 MyGo 会把 `moreOpen`/`dlgOpen` 置为 false，view 检测到后补一次关闭动画。新增覆盖层请沿用该模式。

**日志** — `logs.go` 只读取文件末尾 64KB，并丢弃截断的半行。标准文件顺序是 `<名称>.wrapper.log`、`.out.log`、`.err.log`，之后是其他 `.log`；默认选 wrapper.log。分段控件由 `ui.SegmentedBase` 构建并居中，不存在的文件禁用，一个日志都没有时隐藏；内容区使用固定高度，切换文件不会改变对话框高度。

**国际化** — 主界面的用户可见字符串都来自 `internal/locales`；修改时同时改 `en.json` 和 `zh-Hans.json`，再运行 `go tool mygo generate`。`catalog_gen.go` 是生成文件，不要手改。消息使用 ICU MessageFormat，避免在消息里使用反斜杠。自动更新窗口直接使用 MyGo 内置翻译并跟随 `App.Locale`；`cmd/winswx/updatefeed.go` 包装更新清单，按同一语言选择 `CHANGELOG.md` 中的 `### 中文` 或 `### English` 小节。`updateFeed` 常量必须与 `mygo.json` 的 `updates.github` 对应。

**Windows 细节** — `exec_windows.go` 的 `hideWindow` 使用 `CREATE_NO_WINDOW` + `HideWindow`，防止 WinSW 的控制台窗口闪现；`exec_other.go` 提供非 Windows 的空实现。安装、启动和停止服务需要管理员权限，应用不会自行提权。

**数据目录** — 应用只使用 `%AppData%\winswx`（`bin\winsw.exe` 与 `services\<名称>\`）。旧 Tauri 数据目录 `%AppData%\com.quantulr.winswx` 不做迁移，保持不动。

## 发布

- `.github/workflows/ci.yml`：push/PR 在 windows-latest 上拉取 WinSW，运行 `go vet ./...` 和 `go test ./...`。
- `.github/workflows/release.yml`：推送 `v*` tag 时创建 draft release，并用 `go tool mygo build -platform windows/amd64 -upload` 构建、签名和上传；手动触发只构建未签名 artifact。
- 先改 `mygo.json` 的 `version`，并在 `CHANGELOG.md` 中添加该版本的 `## <version>` 小节，其中用 `### 中文` 和 `### English` 分语言；更新窗口会显示当前语言的小节。更新日志只写用户能感知的更改，重构、依赖升级等内部改动不要写入。缺少版本小节时签名构建会失败。再提交、打 tag（`vX.Y.Z`）并推送。
- 更新签名私钥保存在仓库 secret `MYGO_UPDATER_PRIVATE_KEY` 中，必须与 `mygo.json` 的 `updates.publicKey` 配对；私钥不要提交。仓库 secret 由用户设置。
- 工作流成功后发布 draft：`gh release edit vX.Y.Z --draft=false`。
- 更新源为 `https://github.com/quantulr/winswx/releases/latest/download/update-windows-amd64.json`。
- 应用图标由 `bash design/make-icon.sh <artwork.png>` 生成（需要 ImageMagick），输出到 `resources/icon.png`。
