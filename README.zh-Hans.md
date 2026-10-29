# winswx

一个使用 [MyGo](https://github.com/egoist/mygo) 和
[WinSW](https://github.com/winsw/winsw) 构建的原生 Windows 服务管理器。
用户数据目录下 `services` 的每个子目录就是一个服务，存放该服务的 WinSW
XML 配置。

应用支持英文和中文，默认跟随系统语言；标题栏右侧的地球按钮可以在两者之间
切换。

[English](README.md)

## 构建

应用随包携带的 WinSW 可执行文件不在仓库中：`.gitignore` 忽略了
`resources/bin/winsw.exe`，因此在构建或测试前需要先下载：

```sh
go run ./scripts/fetch-winsw    # 或：go generate ./...
```

脚本会从 WinSW 的 GitHub Release 下载 v3.0.0-alpha.11，校验 SHA-256 后写入
`resources/bin/winsw.exe`；文件已存在时不会重复下载。

然后用 `go.mod` 中固定版本的 MyGo CLI 开发与构建：

```sh
go tool mygo dev
go tool mygo build
```

测试：

```sh
go test ./...
```

## 更新

安装后的应用会从 GitHub 上带签名的 Release 自动更新：启动 10 秒后检查一次，
之后每天检查一次，发现新版本时提示安装。`mygo build` 使用仓库 secret
`MYGO_UPDATER_PRIVATE_KEY` 中的私钥为更新签名，它必须与 `mygo.json` 里的
`updates.publicKey` 配对。请妥善保管该私钥——已安装的应用只会接受用它对
应的密钥签名的更新。在该功能出现之前安装的版本，需要手动安装一次包含它的
版本。

## 数据

应用数据保存在 `%AppData%\winswx`：

```
%AppData%\winswx\
├── bin\
│   └── winsw.exe          安装服务时使用的 wrapper
└── services\
    └── <名称>\
        ├── <名称>.xml     WinSW 配置
        └── *.log          服务日志
```

服务从用户数据目录下的 `bin\winsw.exe` 运行，而不是 app 自身的文件；因此
运行中的服务不会阻止 app 更新或卸载。由旧版本安装的服务注册的是 app 内的
副本，需要重新安装一次以迁移到新路径。

安装、启动和停止服务需要管理员权限，请以管理员身份运行应用。
