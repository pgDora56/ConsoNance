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

## Configuration

Create a `config.yaml` file in the same directory as the executable:

```yaml
discord_token: "YOUR_BOT_TOKEN_HERE"
channel_id: "1234567890"  # Optional: Auto-join this channel on startup
guild_id: "0987654321"    # Your Discord server (guild) ID
audio_device_name: "Your Audio Device Name"  # Optional: Specify audio device
list_devices: false       # Set to true to list available audio devices
```

### Getting Your Discord Bot Token

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application or select an existing one
3. Navigate to the "Bot" section
4. Click "Reset Token" and copy your bot token
5. Paste it into `config.yaml` as `discord_token`

### Finding Guild and Channel IDs

1. Enable Developer Mode in Discord: Settings > Advanced > Developer Mode
2. Right-click on your server icon → Copy Server ID (this is your `guild_id`)
3. Right-click on a voice channel → Copy Channel ID (this is your `channel_id`)

### Audio Device Configuration

To list available audio devices:

```yaml
list_devices: true
```

Then run the bot to see available devices. Copy the device name you want to use and set it in `audio_device_name`.

## Usage

### Starting the Bot

```bash
./consonance  # or consonance.exe on Windows
```

The bot will start and connect to Discord. If `channel_id` is specified in `config.yaml`, it will automatically join that voice channel.

### Discord Commands

You can control the bot by mentioning it in any text channel on your Discord server:

#### Join a Voice Channel

```
@YourBot join #channel-name
```

Or search by channel name (without `#`):

```
@YourBot join channel-name
```

#### Leave Voice Channel

```
@YourBot leave
```

#### Check Status

```
@YourBot status
```

Shows current connection status and streaming information.

#### Help

```
@YourBot help
```

Shows available commands.

### How It Works

- The bot captures system audio (loopback) from your computer
- It encodes the audio to Opus format
- The audio is streamed to the Discord voice channel in real-time
- You can switch channels on-the-fly using Discord commands without restarting the bot

## Notes

- Make sure your Discord bot has the necessary permissions to join voice channels
- Required bot permissions: `Connect`, `Speak`, `Read Messages`, `Send Messages`
- The bot uses loopback audio capture to stream system audio
