# MOP

MOPは **Minecraft Observation Program** の略です。

Minecraft Java Editionサーバーの状態を確認する、Windows向けのシンプルなGUIアプリケーションです。

## 機能

- サーバーのオンライン状態、Ping、バージョン、プレイヤー数を表示
- MOTDとMOD情報を取得
- ポート省略時のMinecraft DNS SRVレコードを自動解決
- 接続結果をクリップボードへコピー
- 接続先とタイムアウト設定を保存

## 開発

Go 1.24以降が必要です。

```powershell
go test ./...
go vet ./...
go run ./cmd/mc-server-checker
```

## Windowsビルド

```powershell
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.0
.\Makefile.ps1 -Version "0.1.0" -Commit "dev"
```

生成物は `output\windows` に出力されます。
