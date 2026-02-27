@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "REPO_OWNER=takubii"
set "REPO_NAME=git-worktree-opener"
set "BINARY_NAME=wto.exe"

if "%~1"=="" (
  echo Usage: %~nx0 vX.Y.Z
  echo Example: %~nx0 v0.2.0
  exit /b 1
)

set "VERSION=%~1"
if /I not "%VERSION:~0,1%"=="v" (
  echo [ERROR] Version must start with "v" ^(example: v0.2.0^).
  exit /b 1
)

if not "%WTO_INSTALL_DIR%"=="" (
  set "INSTALL_DIR=%WTO_INSTALL_DIR%"
) else (
  set "INSTALL_DIR=%USERPROFILE%\bin"
)

set "ARCH_INPUT=%PROCESSOR_ARCHITECTURE%"
if /I "%ARCH_INPUT%"=="x86" if not "%PROCESSOR_ARCHITEW6432%"=="" set "ARCH_INPUT=%PROCESSOR_ARCHITEW6432%"

if /I "%ARCH_INPUT%"=="AMD64" (
  set "ARCH=amd64"
) else if /I "%ARCH_INPUT%"=="ARM64" (
  set "ARCH=arm64"
) else (
  echo [ERROR] Unsupported architecture: %ARCH_INPUT%
  echo Supported architectures: AMD64, ARM64
  exit /b 1
)

where curl >nul 2>nul
if errorlevel 1 (
  echo [ERROR] curl was not found. Install curl and retry.
  exit /b 1
)

where tar >nul 2>nul
if errorlevel 1 (
  echo [ERROR] tar was not found. Install bsdtar/tar and retry.
  exit /b 1
)

where certutil >nul 2>nul
if errorlevel 1 (
  echo [ERROR] certutil was not found. Use Windows built-in certutil and retry.
  exit /b 1
)

set "ASSET_NAME=git-worktree-opener_%VERSION%_windows_%ARCH%.zip"
set "BASE_URL=https://github.com/%REPO_OWNER%/%REPO_NAME%/releases/download/%VERSION%"
set "ARCHIVE_URL=%BASE_URL%/%ASSET_NAME%"
set "CHECKSUMS_URL=%BASE_URL%/checksums.txt"

set "TMP_DIR=%TEMP%\wto-install-%RANDOM%%RANDOM%%RANDOM%"
set "EXTRACT_DIR=%TMP_DIR%\extract"
set "ARCHIVE_PATH=%TMP_DIR%\%ASSET_NAME%"
set "CHECKSUMS_PATH=%TMP_DIR%\checksums.txt"

mkdir "%TMP_DIR%" >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Failed to create temp directory: %TMP_DIR%
  exit /b 1
)

echo Downloading %ASSET_NAME% ...
curl -fsSL -o "%ARCHIVE_PATH%" "%ARCHIVE_URL%"
if errorlevel 1 (
  echo [ERROR] Failed to download archive from:
  echo   %ARCHIVE_URL%
  echo Confirm the tag exists and retry.
  goto :cleanup_fail
)

echo Downloading checksums.txt ...
curl -fsSL -o "%CHECKSUMS_PATH%" "%CHECKSUMS_URL%"
if errorlevel 1 (
  echo [ERROR] Failed to download checksums from:
  echo   %CHECKSUMS_URL%
  goto :cleanup_fail
)

set "EXPECTED_HASH="
for /f "usebackq tokens=1,2" %%A in ("%CHECKSUMS_PATH%") do (
  if /I "%%B"=="%ASSET_NAME%" set "EXPECTED_HASH=%%A"
  if /I "%%B"=="*%ASSET_NAME%" set "EXPECTED_HASH=%%A"
)

if "%EXPECTED_HASH%"=="" (
  echo [ERROR] Could not find checksum entry for %ASSET_NAME%.
  goto :cleanup_fail
)

set "ACTUAL_HASH="
for /f "usebackq delims=" %%H in (`certutil -hashfile "%ARCHIVE_PATH%" SHA256 ^| findstr /R /I "^[0-9A-F][0-9A-F]*$"`) do (
  set "ACTUAL_HASH=%%H"
  goto :hash_done
)

:hash_done
set "ACTUAL_HASH=%ACTUAL_HASH: =%"
if "%ACTUAL_HASH%"=="" (
  echo [ERROR] Failed to parse computed SHA256 hash.
  goto :cleanup_fail
)

if /I not "%ACTUAL_HASH%"=="%EXPECTED_HASH%" (
  echo [ERROR] Checksum mismatch for %ASSET_NAME%.
  echo Expected: %EXPECTED_HASH%
  echo Actual:   %ACTUAL_HASH%
  goto :cleanup_fail
)

mkdir "%EXTRACT_DIR%" >nul 2>nul
tar -xf "%ARCHIVE_PATH%" -C "%EXTRACT_DIR%"
if errorlevel 1 (
  echo [ERROR] Failed to extract archive: %ARCHIVE_PATH%
  goto :cleanup_fail
)

set "BIN_PATH="
if exist "%EXTRACT_DIR%\%BINARY_NAME%" (
  set "BIN_PATH=%EXTRACT_DIR%\%BINARY_NAME%"
) else (
  for /r "%EXTRACT_DIR%" %%F in (%BINARY_NAME%) do (
    set "BIN_PATH=%%F"
    goto :bin_found
  )
)

:bin_found
if "%BIN_PATH%"=="" (
  echo [ERROR] Downloaded archive does not contain %BINARY_NAME%.
  goto :cleanup_fail
)

if not exist "%INSTALL_DIR%\" (
  mkdir "%INSTALL_DIR%" >nul 2>nul
  if errorlevel 1 (
    echo [ERROR] Failed to create install directory: %INSTALL_DIR%
    goto :cleanup_fail
  )
)

copy /y "%BIN_PATH%" "%INSTALL_DIR%\%BINARY_NAME%" >nul
if errorlevel 1 (
  echo [ERROR] Failed to copy binary to install directory.
  goto :cleanup_fail
)

echo Installed %BINARY_NAME% %VERSION% to %INSTALL_DIR%

echo ;%PATH%; | findstr /I /C:";%INSTALL_DIR%;" >nul
if errorlevel 1 set "PATH=%INSTALL_DIR%;%PATH%"

echo wto is ready in this cmd.exe session.
echo To persist PATH for future sessions, add this directory manually:
echo   %INSTALL_DIR%

goto :cleanup_success

:cleanup_fail
set "EXIT_CODE=1"
goto :cleanup

:cleanup_success
set "EXIT_CODE=0"
goto :cleanup

:cleanup
if exist "%TMP_DIR%" rmdir /s /q "%TMP_DIR%" >nul 2>nul
exit /b %EXIT_CODE%
