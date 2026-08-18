param(
    [string]$Version = "0.1.1",
    [string]$Commit = "dev"
)

$ErrorActionPreference = "Stop"
$projectRoot = $PSScriptRoot
$outputDirectory = Join-Path $projectRoot "output\windows"
$outputName = "mc-server-checker_v${Version}_windows_amd64.exe"
$outputPath = Join-Path $outputDirectory $outputName
$manifestSource = Join-Path $projectRoot "build\windows\app.manifest"
$manifestGenerated = Join-Path $projectRoot "build\windows\app.generated.manifest"
$resourcePath = Join-Path $projectRoot "cmd\mc-server-checker\resource_windows_amd64.syso"
$versionInfo = Join-Path $projectRoot "build\windows\versioninfo.json"
$icon = Join-Path $projectRoot "assets\icon.ico"
$goversioninfo = Join-Path (go env GOPATH) "bin\goversioninfo.exe"
$buildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

if (-not (Test-Path -LiteralPath $goversioninfo)) {
    throw "goversioninfo was not found. Install the pinned tool with: go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.0"
}

$versionParts = $Version.Split(".")
if ($versionParts.Count -ne 3) { throw "Version must use x.y.z format" }
$major = [int]$versionParts[0]
$minor = [int]$versionParts[1]
$patch = [int]$versionParts[2]
$ldflags = "-H=windowsgui -s -w -X mc-server-checker/internal/platform.Version=$Version -X mc-server-checker/internal/platform.Commit=$Commit -X mc-server-checker/internal/platform.BuildDate=$buildDate"

New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null

Push-Location $projectRoot
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failed" }

    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed" }

    $manifest = Get-Content -Raw -LiteralPath $manifestSource
    $assemblyVersion = [regex]'version="[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"'
    $assemblyVersion.Replace($manifest, "version=`"$Version.1`"", 1) | Set-Content -Encoding UTF8 -LiteralPath $manifestGenerated
    try {
        & $goversioninfo -64 -o $resourcePath -icon $icon -manifest $manifestGenerated -file-version $Version -product-version $Version -original-name $outputName -ver-major $major -ver-minor $minor -ver-patch $patch -ver-build 1 -product-ver-major $major -product-ver-minor $minor -product-ver-patch $patch -product-ver-build 1 $versionInfo
        if ($LASTEXITCODE -ne 0) { throw "goversioninfo failed" }

        go build -trimpath -ldflags $ldflags -o $outputPath ./cmd/mc-server-checker
        if ($LASTEXITCODE -ne 0) { throw "go build failed" }
    }
    finally {
        if (Test-Path -LiteralPath $resourcePath) {
            [System.IO.File]::Delete($resourcePath)
        }
        if (Test-Path -LiteralPath $manifestGenerated) {
            [System.IO.File]::Delete($manifestGenerated)
        }
    }

    $hash = Get-FileHash -Algorithm SHA256 -LiteralPath $outputPath
    Set-Content -Encoding ASCII -NoNewline -LiteralPath "$outputPath.sha256" -Value ($hash.Hash + "  " + $outputName)
    Write-Output $outputPath
    Write-Output ("SHA256: " + $hash.Hash)
}
finally {
    Pop-Location
}
