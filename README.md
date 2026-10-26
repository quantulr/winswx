# winswx

A native Windows service manager built with [MyGo](https://github.com/egoist/mygo)
and [WinSW](https://github.com/winsw/winsw). Each directory under the user
data's `services` directory is a service, and holds its WinSW XML
configuration.

The app shows in English or Chinese, following the system language; the
globe button in the title bar switches between them.

## Build

The WinSW executable the app ships is not in the repository: `.gitignore`
keeps `resources/bin/winsw.exe` out, so fetch it before building or testing:

```sh
go run ./scripts/fetch-winsw    # or: go generate ./...
```

The script downloads WinSW v3.0.0-alpha.11 from its GitHub release, checks
its SHA-256 and writes it to `resources/bin/winsw.exe`. It does nothing
when the file is already there.

Then develop and build with the MyGo CLI pinned in `go.mod`:

```sh
go tool mygo dev
go tool mygo build
```

Tests:

```sh
go test ./...
```

## Data

The app keeps its data in `%AppData%\winswx`:

```
%AppData%\winswx\
└── services\
    └── <name>\
        ├── <name>.xml    the WinSW configuration
        └── *.log         the service's logs
```

Installing, starting and stopping services needs administrator rights; run
the app elevated for those operations.
