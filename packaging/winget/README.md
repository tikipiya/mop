# WinGet manifest

`manifests`以下は、Microsoft Community Package Manifest Repositoryへ提出するためのファイルです。

パッケージ識別子は`tikipiya.MOP`、インストールコマンドは次のとおりです。

```powershell
winget install --id tikipiya.MOP --exact
```

## 新しいバージョンの追加

1. GitHub Releaseへバージョン付きWindows EXEを公開する。
2. 前バージョンのディレクトリを新しいバージョン番号で複製する。
3. すべての`PackageVersion`、リリースURL、リリースノートURL、`ReleaseDate`を更新する。
4. 公開したEXEのSHA-256を取得して`InstallerSha256`を更新する。
5. manifestを検証する。

```powershell
winget hash output\windows\mc-server-checker_vX.Y.Z_windows_amd64.exe
winget validate --manifest packaging\winget\manifests\t\tikipiya\MOP\X.Y.Z
```

6. `manifests\t\tikipiya\MOP\X.Y.Z`を`microsoft/winget-pkgs`の同じパスへ追加し、1バージョンだけを含むPRを作成する。
