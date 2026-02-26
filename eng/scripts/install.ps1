param (
    [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"

$Repo = "frostyeti/cast"
$ApiUrl = if ($Version -eq "latest") { "https://api.github.com/repos/$Repo/releases/latest" } else { "https://api.github.com/repos/$Repo/releases/tags/$Version" }

# Determine OS
$OsName = "Unknown"
if ($IsWindows -or [System.Environment]::OSVersion.Platform -eq "Win32NT") {
    $OsName = "Windows"
} elseif ($IsLinux) {
    $OsName = "Linux"
} elseif ($IsMacOS) {
    $OsName = "Darwin"
} else {
    Write-Error "Unsupported Operating System"
}

# Determine Architecture
$ArchName = "Unknown"
$Arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
if ($Arch -eq [System.Runtime.InteropServices.Architecture]::X64) {
    $ArchName = "x86_64"
} elseif ($Arch -eq [System.Runtime.InteropServices.Architecture]::Arm64) {
    $ArchName = "arm64"
} elseif ($Arch -eq [System.Runtime.InteropServices.Architecture]::X86) {
    $ArchName = "i386"
} else {
    Write-Error "Unsupported Architecture: $Arch"
}

# Determine Install Directory
$InstallDir = $env:CAST_INSTALL_DIR
if ([string]::IsNullOrEmpty($InstallDir)) {
    if ($OsName -eq "Windows") {
        $InstallDir = Join-Path $env:USERPROFILE "AppData\Local\Programs\bin"
    } else {
        $InstallDir = Join-Path $env:HOME ".local\bin"
    }
}

if (-not (Test-Path -Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

$Ext = if ($OsName -eq "Windows") { "zip" } else { "tar.gz" }

Write-Host "Fetching release information for $Repo ($Version)..."
try {
    $Release = Invoke-RestMethod -Uri $ApiUrl -Headers @{"Accept"="application/vnd.github.v3+json"}
} catch {
    Write-Error "Failed to fetch release info: $_"
}

$AssetName = "cast_${OsName}_${ArchName}.$Ext"
$Asset = $Release.assets | Where-Object { $_.name -eq $AssetName }

if (-not $Asset) {
    Write-Error "Could not find a release asset matching $AssetName"
}

$DownloadUrl = $Asset.browser_download_url
$TmpDir = Join-Path [System.IO.Path]::GetTempPath() ([guid]::NewGuid().ToString())
New-Item -ItemType Directory -Force -Path $TmpDir | Out-Null
$TmpFile = Join-Path $TmpDir $AssetName

Write-Host "Downloading Cast from $DownloadUrl..."
Invoke-WebRequest -Uri $DownloadUrl -OutFile $TmpFile

Write-Host "Extracting..."
if ($Ext -eq "zip") {
    Expand-Archive -Path $TmpFile -DestinationPath $TmpDir -Force
} else {
    # Cross platform tar extraction (requires tar to be in path, typically available in modern Windows/Linux/Mac)
    & tar -xzf $TmpFile -C $TmpDir
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to extract tar.gz archive"
    }
}

$ExecutableName = if ($OsName -eq "Windows") { "cast.exe" } else { "cast" }
$ExtractedFile = Join-Path $TmpDir $ExecutableName
$DestFile = Join-Path $InstallDir $ExecutableName

Move-Item -Path $ExtractedFile -Destination $DestFile -Force

if ($OsName -ne "Windows") {
    # Make executable on Linux/Mac
    & chmod +x $DestFile
}

Remove-Item -Path $TmpDir -Recurse -Force

Write-Host "Cast installed to $DestFile"

# Check PATH
$PathArray = if ($OsName -eq "Windows") { $env:PATH -split ';' } else { $env:PATH -split ':' }
if ($InstallDir -notin $PathArray) {
    Write-Host "================================================================================" -ForegroundColor Yellow
    Write-Host "WARNING: $InstallDir is not in your PATH." -ForegroundColor Yellow
    if ($OsName -eq "Windows") {
        Write-Host "Please add it to your System or User Environment Variables." -ForegroundColor Yellow
    } else {
        Write-Host "Please add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):" -ForegroundColor Yellow
        Write-Host "export PATH=`"`$PATH:$InstallDir`"" -ForegroundColor Yellow
    }
    Write-Host "================================================================================" -ForegroundColor Yellow
}

Write-Host "Installation complete! Run 'cast --help' to get started." -ForegroundColor Green
