# ConsoNance

Consonance <-> Discord

Audio Stream Bot for Discord

## Overview

WIP

## Prerequisites

- Go 1.16 or later
- CGO-enabled environment
- C compiler (gcc)
- Opus library

## Building

### Windows

#### Requirements

To build on Windows, you need to install MSYS2 and required packages:

1. **Install MSYS2**
   - Download the installer from https://www.msys2.org/
   - Run the installer and follow the default installation (installs to `C:\msys64`)

2. **Update MSYS2**
   - Open "MSYS2 MSYS" from the Start Menu
   - Run the following command:
     ```bash
     pacman -Syu
     ```
   - The terminal will close after the update. Open "MSYS2 MSYS" again and run:
     ```bash
     pacman -Su
     ```

3. **Install required packages**
   - Open "MSYS2 MinGW64" from the Start Menu
   - Run the following command:
     ```bash
     pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-pkg-config mingw-w64-x86_64-opus
     ```

4. **Set environment variables**
   - In PowerShell, add MinGW64 to your PATH:
     ```powershell
     $env:PATH += ";C:\msys64\mingw64\bin"
     $env:CGO_ENABLED = "1"
     ```

5. **Build the application**
   ```powershell
   go build -o consonance-win.exe
   ```
   
   Or using Makefile:
   ```powershell
   make build-windows
   ```

### Linux

Ensure you have gcc and opus library installed:

```bash
sudo apt-get install gcc libopus-dev  # For Debian/Ubuntu
# or
sudo yum install gcc opus-devel       # For CentOS/RHEL
```

Then build:

```bash
go build -o consonance
```

Or using Makefile:

```bash
make build-linux
```

### macOS

Install opus library using Homebrew:

```bash
brew install opus
```

Then build:

```bash
go build -o consonance
```

Or using Makefile:

```bash
make build-mac
```

## Usage

```bash
./consonance  # or consonance-win.exe on Windows
```
