# 🔐 UnFuckable USB

Portable USB encryption tool. Your data, your rules.

## What it does

Encrypts USB drives so only you can access them. Works on Windows, Linux, macOS.

## Features

- **AES-256-GCM + XChaCha20** — double-layer encryption
- **Chunk storage** — data splits into random-sized pieces with garbage names
- **Panic button** — Ctrl+Shift+F12 encrypts everything instantly
- **Decoy files** — encrypted data looks like temp files
- **No installation** — single portable executable

## Download

Get the latest release for your platform from [Releases](../../releases).

## Usage

1. Run the executable
2. Select your USB drive
3. Set a password
4. Done

Press `Ctrl+Shift+F12` anytime to panic-encrypt all decrypted drives.

## Build
```bash
./build.sh      # Linux/macOS
build.bat       # Windows
```

Requires Go 1.21+

## Languages

English, Русский, Українська

## License

MIT

---

*Making your data impossible to fuck with.*
