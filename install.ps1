<#
.SYNOPSIS
Install stk (Sigma Token Killer) on Windows.

.PARAMETER Version
Release tag to install (example: v0.1.0). Default: latest. Env: VERSION.

.PARAMETER FromFile
Path to a local release archive (.zip). Skips the download. Env: STK_FROM_FILE.

.PARAMETER ChecksumsFile
Path to a local checksums.txt for -FromFile. Env: STK_CHECKSUMS_FILE.

.PARAMETER InstallDir
Install directory. Default: $env:LOCALAPPDATA\Programs\stk. Env: INSTALL_DIR.
#>
param(
    [string]$Version = $env:VERSION,
    [string]$FromFile = $env:STK_FROM_FILE,
    [string]$ChecksumsFile = $env:STK_CHECKSUMS_FILE,
    [string]$InstallDir = $env:INSTALL_DIR
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
# Windows PowerShell 5.1 needs TLS 1.2 for GitHub.
try { [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 } catch { }
$Repo = 'sigmashakeinc/stk'

if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\stk'
}

$archRaw = $env:PROCESSOR_ARCHITEW6432
if (-not $archRaw) { $archRaw = $env:PROCESSOR_ARCHITECTURE }
switch ($archRaw) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    default { throw "Unsupported architecture: $archRaw" }
}

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("stk-" + [System.Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    if ($FromFile) {
        if (-not (Test-Path -LiteralPath $FromFile)) { throw "Archive not found: $FromFile" }
        $archive = (Resolve-Path -LiteralPath $FromFile).Path
        $archiveName = Split-Path -Leaf $archive
        if ($ChecksumsFile) {
            if (-not (Test-Path -LiteralPath $ChecksumsFile)) { throw "Checksums file not found: $ChecksumsFile" }
            $sums = (Resolve-Path -LiteralPath $ChecksumsFile).Path
        } else {
            $sums = $null
            Write-Warning 'No checksums file given. Skipping verification of the local archive.'
        }
    } else {
        if (-not $Version) {
            # The /releases/latest URL redirects to /releases/tag/<tag>.
            # Do not follow the redirect. Read the Location header.
            # This works on Windows PowerShell 5.1 and on PowerShell 7.
            $req = [Net.WebRequest]::Create("https://github.com/$Repo/releases/latest")
            $req.AllowAutoRedirect = $false
            $resp = $req.GetResponse()
            try { $loc = $resp.Headers['Location'] } finally { $resp.Close() }
            if (-not $loc) { throw 'Cannot resolve the latest release: no Location header.' }
            $Version = ($loc -split '/tag/')[-1]
        }
        $archiveName = "stk_${Version}_windows_${arch}.zip"
        $base = "https://github.com/$Repo/releases/download/$Version"
        $archive = Join-Path $tmp $archiveName
        $sums = Join-Path $tmp 'checksums.txt'
        Write-Host "Downloading $archiveName"
        Invoke-WebRequest -Uri "$base/$archiveName" -OutFile $archive -UseBasicParsing
        Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $sums -UseBasicParsing
    }

    if ($sums) {
        $expected = $null
        foreach ($line in Get-Content -LiteralPath $sums) {
            $parts = $line -split '\s+', 2
            if ($parts.Count -eq 2 -and ($parts[1].TrimStart('*') -eq $archiveName)) {
                $expected = $parts[0].ToLowerInvariant()
                break
            }
        }
        if (-not $expected) { throw "No checksum entry for $archiveName" }
        $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
        if ($actual -ne $expected) {
            throw "Checksum mismatch for $archiveName (expected $expected, got $actual)"
        }
        Write-Host 'Checksum OK'
    }

    $extract = Join-Path $tmp 'x'
    Expand-Archive -LiteralPath $archive -DestinationPath $extract -Force
    $exe = Join-Path $extract 'stk.exe'
    if (-not (Test-Path -LiteralPath $exe)) { throw 'Archive has no stk.exe.' }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -LiteralPath $exe -Destination (Join-Path $InstallDir 'stk.exe') -Force
    Write-Host "Installed stk to $(Join-Path $InstallDir 'stk.exe')"

    # Add the install dir to the user PATH. No admin needed.
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not $userPath) { $userPath = '' }
    $entries = @($userPath -split ';' | Where-Object { $_ -ne '' })
    if ($entries -notcontains $InstallDir) {
        $newPath = (($entries + $InstallDir) -join ';')
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        Write-Host "Added $InstallDir to your user PATH. Open a new terminal to use stk."
    }
    if (($env:Path -split ';') -notcontains $InstallDir) {
        $env:Path = "$env:Path;$InstallDir"
    }
} finally {
    Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
