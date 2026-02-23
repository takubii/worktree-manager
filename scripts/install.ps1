Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoOwner = "takubii"
$repoName = "git-worktree-opener"
$binaryName = "wto.exe"

$version = $env:WTO_VERSION
if ([string]::IsNullOrWhiteSpace($version)) {
  $version = ""
}

$installDir = $env:WTO_INSTALL_DIR
if ([string]::IsNullOrWhiteSpace($installDir)) {
  $installDir = Join-Path $HOME "bin"
}

$skipChecksum = ($env:WTO_SKIP_CHECKSUM -eq "1")

function Resolve-LatestVersion {
  $apiUrl = "https://api.github.com/repos/$repoOwner/$repoName/releases/latest"
  $release = Invoke-RestMethod -Uri $apiUrl
  if ([string]::IsNullOrWhiteSpace($release.tag_name)) {
    throw "Failed to resolve latest release version from GitHub API."
  }
  return [string]$release.tag_name
}

function Resolve-Arch {
  $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
  switch ($arch.ToString().ToLowerInvariant()) {
    "x64" { return "amd64" }
    "arm64" { return "arm64" }
    default { throw "Unsupported architecture: $arch. Supported: amd64, arm64." }
  }
}

function Resolve-ExpectedChecksum {
  param (
    [Parameter(Mandatory = $true)][string]$ChecksumsPath,
    [Parameter(Mandatory = $true)][string]$ArchiveName
  )

  $line = Get-Content $ChecksumsPath | Where-Object {
    $_ -match "^(?<hash>[0-9a-fA-F]{64})\s+\*?(?<name>.+)$" -and $Matches.name -eq $ArchiveName
  } | Select-Object -First 1

  if ([string]::IsNullOrWhiteSpace($line)) {
    throw "Checksum entry not found for $ArchiveName."
  }

  $null = $line -match "^(?<hash>[0-9a-fA-F]{64})\s+\*?(?<name>.+)$"
  return $Matches.hash.ToLowerInvariant()
}

if ([string]::IsNullOrWhiteSpace($version)) {
  $version = Resolve-LatestVersion
}

$arch = Resolve-Arch
$archiveName = "git-worktree-opener_${version}_windows_${arch}.zip"
$archiveUrl = "https://github.com/$repoOwner/$repoName/releases/download/$version/$archiveName"
$checksumsUrl = "https://github.com/$repoOwner/$repoName/releases/download/$version/checksums.txt"

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("wto-install-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
  $archivePath = Join-Path $tempDir $archiveName
  $checksumsPath = Join-Path $tempDir "checksums.txt"
  $extractDir = Join-Path $tempDir "extract"

  Write-Host "Downloading $archiveName ..."
  Invoke-WebRequest -Uri $archiveUrl -OutFile $archivePath
  Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath

  if (-not $skipChecksum) {
    $expected = Resolve-ExpectedChecksum -ChecksumsPath $checksumsPath -ArchiveName $archiveName
    $actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
      throw "Checksum mismatch for $archiveName."
    }
  }

  Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force
  $binary = Get-ChildItem -Path $extractDir -Filter $binaryName -File -Recurse | Select-Object -First 1
  if ($null -eq $binary) {
    throw "Downloaded archive does not contain $binaryName."
  }

  New-Item -ItemType Directory -Path $installDir -Force | Out-Null
  Copy-Item -Path $binary.FullName -Destination (Join-Path $installDir $binaryName) -Force

  Write-Host "Installed $binaryName $version to $installDir"

  $pathEntries = @($env:Path -split ";") | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
  $inPath = $pathEntries | Where-Object { $_.TrimEnd("\") -ieq $installDir.TrimEnd("\") }
  if (-not $inPath) {
    Write-Host "Add this directory to PATH to run wto from any terminal:"
    Write-Host "  $installDir"
  }
} finally {
  Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
