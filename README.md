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

### Easy First-Time Setup

**No manual configuration required!** Just run the bot:

```bash
./consonance.exe  # or ./consonance on Linux/macOS
```

On first run, the bot will:
1. Automatically create `config.yaml` if it doesn't exist
2. Prompt you to enter your Discord bot token
3. Save your token to `config.yaml`
4. Prompt you to select an audio device
5. Optionally save the device as default

Example first run:
```
=== Discord Bot Token Required ===
Your Discord bot token is not configured.

To get your bot token:
1. Go to https://discord.com/developers/applications
2. Select your application (or create a new one)
3. Navigate to the 'Bot' section
4. Click 'Reset Token' or 'Copy' to get your token
5. Make sure to enable 'MESSAGE CONTENT INTENT' under Privileged Gateway Intents

Paste your Discord bot token here: [YOUR_TOKEN]
✓ Discord token saved to config.yaml

=== Select Audio Device ===
Available audio devices for loopback capture:

[1] Speakers (Realtek High Definition Audio) (Default)
[2] Line (Yamaha SYNCROOM Driver)

Enter device number: 2
✓ Selected: Line (Yamaha SYNCROOM Driver)

Save this device as default in config.yaml? (y/n): y
✓ Device saved to config.yaml
```

### Manual Configuration (Optional)

If you prefer to manually configure, copy `config.yaml.example` to `config.yaml`:

```bash
cp config.yaml.example config.yaml
```

Minimum required configuration:

```yaml
discord_token: "YOUR_BOT_TOKEN_HERE"
```

### Optional Auto-Connect Settings

To automatically join a voice channel on startup:

```yaml
discord_token: "YOUR_BOT_TOKEN_HERE"
guild_id: "0987654321"    # Your Discord server (guild) ID
channel_id: "1234567890"  # Voice channel to auto-join
```

### Getting Your Discord Bot Token

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application or select an existing one
3. Navigate to the "Bot" section
4. Click "Reset Token" and copy your bot token

### Important: Enable Privileged Gateway Intents

⚠️ **This step is required or you'll get authentication errors!**

1. In the Discord Developer Portal, go to your application
2. Navigate to the **"Bot"** section
3. Scroll down to **"Privileged Gateway Intents"**
4. Enable the following intents:
   - ✅ **MESSAGE CONTENT INTENT** (required for commands)
   - ✅ **SERVER MEMBERS INTENT** (recommended)
   - ✅ **PRESENCE INTENT** (recommended)
5. Click **"Save Changes"**

Without these intents enabled, you'll see an error like:
```
websocket: close 4004: Authentication failed.
```

### Audio Device Selection

The bot supports two ways to select an audio device:

1. **Interactive Selection (Recommended)**: Leave `audio_device_name` empty or commented out in `config.yaml`. When you start the bot:
   - You'll see a numbered list of available devices
   - Enter the device number to select it
   - Optionally save your selection to `config.yaml` for future use

   ```
   === Select Audio Device ===
   Available audio devices for loopback capture:
   
   [1] Speakers (Realtek High Definition Audio) (Default)
   [2] Line (Yamaha SYNCROOM Driver)
   [3] Headphones (USB Audio)
   
   Enter device number: 2
   ✓ Selected: Line (Yamaha SYNCROOM Driver)
   
   Save this device as default in config.yaml? (y/n): y
   ✓ Device saved to config.yaml
   ```

2. **Pre-configured Device**: Set `audio_device_name` in `config.yaml` to use a specific device automatically:
   ```yaml
   audio_device_name: "Line (Yamaha SYNCROOM Driver)"
   ```

## Usage

### Starting the Bot

Simply run the executable:

```bash
./consonance.exe  # on Windows
./consonance      # on Linux/macOS
```

**First-time users**: The bot will guide you through:
1. Creating `config.yaml` (if it doesn't exist)
2. Entering your Discord bot token
3. Selecting an audio device

**Subsequent runs**: The bot will use your saved configuration and start immediately.

The bot will connect to Discord. You can then:
- Use Discord chat commands to join/leave voice channels (see below)
- Or, if `guild_id` and `channel_id` are set in `config.yaml`, it will auto-join that channel

### Discord Commands

You can control the bot by mentioning it in any text channel on your Discord server.

**No configuration required!** Just mention the bot with commands:

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

## Development

### Version Management

ConsoNance uses a simple version management system with the format `YY.MM.minor` (e.g., `25.12.1`):

- **YY**: Year (2-digit)
- **MM**: Month (2-digit)
- **minor**: Minor version number within the month

To update the version:

1. Edit the `Version` constant in `version.go`:
   ```go
   const Version = "25.12.2"  // Update this value
   ```

2. The version will be displayed:
   - On application startup (in logs and console)
   - In the `@Bot help` command response

### Build Tags

The project includes several build targets in the `Makefile`:

- `make build-win`: Build for Windows (PowerShell)
- `make build-linux`: Build for Linux
- `make build-mac`: Build for macOS

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

### What does GPL v3.0 mean?

- ✅ You can use this software freely
- ✅ You can modify the source code
- ✅ You can distribute modified versions
- ⚠️ Any derivative work must also be licensed under GPL v3.0
- ⚠️ You must disclose the source code of derivative works
- ⚠️ You must include the original copyright and license notices

Copyright (C) 2025 Kazuki F.
