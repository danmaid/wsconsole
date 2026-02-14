param(
    [ValidateSet("amd64", "arm64", "armv7", "all")] 
    [string]$Arch = "amd64",
    [string]$Output = "wsconsole"
)

function Set-GoEnv {
    param(
        [string]$GoArch,
        [string]$GoArm
    )
    $env:GOOS = "linux"
    $env:GOARCH = $GoArch
    if ($GoArm) {
        $env:GOARM = $GoArm
    } else {
        if (Test-Path Env:GOARM) { Remove-Item Env:GOARM }
    }
}

function Build-Binary {
    param(
        [string]$GoArch,
        [string]$GoArm,
        [string]$Suffix,
        [string]$OutName
    )
    Set-GoEnv -GoArch $GoArch -GoArm $GoArm
    $out = if ($OutName) { $OutName } else { "wsconsole-linux-$Suffix" }
    Write-Host "Building $out (GOARCH=$GoArch GOARM=$GoArm)" -ForegroundColor Cyan
    go build -ldflags "-s -w" -o $out ./cmd/wsconsole
}

switch ($Arch) {
    "amd64" { Build-Binary -GoArch "amd64" -GoArm "" -Suffix "amd64" -OutName $Output }
    "arm64" { Build-Binary -GoArch "arm64" -GoArm "" -Suffix "arm64" -OutName $Output }
    "armv7" { Build-Binary -GoArch "arm" -GoArm "7" -Suffix "armv7" -OutName $Output }
    "all" {
        Build-Binary -GoArch "amd64" -GoArm "" -Suffix "amd64"
        Build-Binary -GoArch "arm64" -GoArm "" -Suffix "arm64"
        Build-Binary -GoArch "arm" -GoArm "7" -Suffix "armv7"
    }
}

Write-Host "Done." -ForegroundColor Green
Write-Host "Note: deb packaging should be run on Linux (or via Docker)." -ForegroundColor Yellow
