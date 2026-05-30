Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoOwner = "takubii"
$repoName = "worktree-manager"
$binaryName = "wtm.exe"

$version = $env:WTM_VERSION
if ([string]::IsNullOrWhiteSpace($version)) {
  $version = ""
}

$installDir = $env:WTM_INSTALL_DIR
if ([string]::IsNullOrWhiteSpace($installDir)) {
  $installDir = Join-Path $HOME "bin"
}

$skipChecksum = ($env:WTM_SKIP_CHECKSUM -eq "1")

function Resolve-Release {
  param(
    [Parameter(Mandatory = $false)][string]$RequestedVersion
  )

  $apiUrl = "https://api.github.com/repos/$repoOwner/$repoName/releases/latest"
  if (-not [string]::IsNullOrWhiteSpace($RequestedVersion)) {
    $apiUrl = "https://api.github.com/repos/$repoOwner/$repoName/releases/tags/$RequestedVersion"
  }

  try {
    return Invoke-RestMethod -Uri $apiUrl
  } catch {
    if ([string]::IsNullOrWhiteSpace($RequestedVersion)) {
      throw "Failed to resolve latest release metadata from GitHub API."
    }
    throw "Failed to resolve release metadata for tag '$RequestedVersion'. Confirm the tag exists and retry."
  }
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

function Normalize-PathEntry {
  param([Parameter(Mandatory = $true)][string]$PathEntry)

  $trimmed = $PathEntry.Trim()
  if ([string]::IsNullOrWhiteSpace($trimmed)) {
    return ""
  }

  try {
    return [System.IO.Path]::GetFullPath($trimmed).TrimEnd("\").ToLowerInvariant()
  } catch {
    return $trimmed.TrimEnd("\").ToLowerInvariant()
  }
}

function Split-PathEntries {
  param([Parameter(Mandatory = $false)][string]$PathValue)

  if ([string]::IsNullOrWhiteSpace($PathValue)) {
    return @()
  }

  return @($PathValue -split ";" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}

function Contains-PathEntry {
  param(
    [Parameter(Mandatory = $false)][string]$PathValue,
    [Parameter(Mandatory = $true)][string]$Candidate
  )

  $normalizedCandidate = Normalize-PathEntry -PathEntry $Candidate
  if ([string]::IsNullOrWhiteSpace($normalizedCandidate)) {
    return $false
  }

  foreach ($entry in Split-PathEntries -PathValue $PathValue) {
    if ((Normalize-PathEntry -PathEntry $entry) -eq $normalizedCandidate) {
      return $true
    }
  }

  return $false
}

function Append-PathEntry {
  param(
    [Parameter(Mandatory = $false)][string]$PathValue,
    [Parameter(Mandatory = $true)][string]$Entry
  )

  if ([string]::IsNullOrWhiteSpace($PathValue)) {
    return $Entry
  }

  $trimmed = $PathValue.TrimEnd(";")
  if ([string]::IsNullOrWhiteSpace($trimmed)) {
    return $Entry
  }

  return "$trimmed;$Entry"
}

$arch = Resolve-Arch
$release = Resolve-Release -RequestedVersion $version
$version = [string]$release.tag_name
if ([string]::IsNullOrWhiteSpace($version)) {
  throw "Release metadata does not include tag_name."
}

$archivePattern = "^worktree-manager_.+_windows_${arch}\.zip$"
$archiveAsset = $release.assets | Where-Object { $_.name -match $archivePattern } | Select-Object -First 1
if ($null -eq $archiveAsset) {
  throw "No Windows archive asset found for architecture '$arch'."
}

$checksumsAsset = $release.assets | Where-Object { $_.name -eq "checksums.txt" } | Select-Object -First 1
if ($null -eq $checksumsAsset) {
  throw "checksums.txt asset was not found in the selected release."
}

$archiveName = [string]$archiveAsset.name
$archiveUrl = [string]$archiveAsset.browser_download_url
$checksumsUrl = [string]$checksumsAsset.browser_download_url
if ([string]::IsNullOrWhiteSpace($archiveUrl)) {
  throw "Archive download URL is missing in release metadata."
}
if ([string]::IsNullOrWhiteSpace($checksumsUrl)) {
  throw "Checksums download URL is missing in release metadata."
}

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("wtm-install-" + [Guid]::NewGuid().ToString("N"))
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

  $installDir = [System.IO.Path]::GetFullPath($installDir)

  Write-Host "Installed $binaryName $version to $installDir"

  if (-not (Contains-PathEntry -PathValue $env:Path -Candidate $installDir)) {
    $env:Path = Append-PathEntry -PathValue $env:Path -Entry $installDir
  }

  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if (-not (Contains-PathEntry -PathValue $userPath -Candidate $installDir)) {
    $updatedUserPath = Append-PathEntry -PathValue $userPath -Entry $installDir
    [Environment]::SetEnvironmentVariable("Path", $updatedUserPath, "User")
    Write-Host "Added $installDir to User PATH."
  }

  $wtmCommand = Get-Command "wtm" -ErrorAction SilentlyContinue
  if ($null -ne $wtmCommand) {
    Write-Host "wtm is ready in this terminal."
  } else {
    Write-Host "wtm command is not available yet in this terminal."
    Write-Host "Run directly once: $installDir\\wtm.exe --help"
    Write-Host "If needed, open a new terminal and run: wtm --help"
  }
} finally {
  Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
