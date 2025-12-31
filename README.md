<img width="1520" height="660" alt="image" src="https://github.com/user-attachments/assets/096e2102-2bdb-4426-aacc-c4ff889c92c1" />

# 🔐 UnFuckable USB

Portable USB encryption tool. Your data, your rules.

## What it does

Encrypts USB drives so only you can access them. Works on Windows, Linux, macOS.

Your encrypted drive will look like Windows barfed temp files all over it. Perfect stealth.

## Features

- **AES-256-GCM + XChaCha20** — double-layer encryption (because one layer is for amateurs)
- **Chunk storage** — data splits into random-sized pieces with garbage names
- **Panic button** — Ctrl+Shift+F12 encrypts everything instantly (for when the feds knock)
- **Decoy files** — encrypted data looks like temp files nobody wants to open
- **No installation** — single portable executable (drag and drop, that's it)

## Download

Get the latest release for your platform from [Releases](../../releases).

## Usage

1. Run the executable
2. Select your USB drive
3. Set a password (write it down unless you have perfect memory)
4. Done

Press `Ctrl+Shift+F12` anytime to panic-encrypt all decrypted drives.

## How it works

**Encryption:**
1. Compresses your files (tar.gz)
2. Encrypts with AES-256-GCM
3. Encrypts again with XChaCha20-Poly1305 (double tap for good measure)
4. Splits into random chunks (1-50 MB each)
5. Renames chunks to look like temp files (`.tmp`, `.log`, `.cache`, `~$garbage`)
6. Adds HMAC to each chunk for integrity
7. Generates 50-200 decoy files (more trash to blend in)
8. Securely wipes original files (3-pass overwrite — they're gone for good)

**Result:** Your USB looks like it's full of random system junk. Government agents will think you just have a dirty filesystem. Even your tech-savvy friend won't notice anything suspicious.

**Decryption:**
1. Reads all chunks
2. Verifies HMAC integrity (catches tampering)
3. Reassembles and decrypts data
4. Restores your files (like magic, but with math)

## Security

- **Argon2id** key derivation (1 GB memory, 4 iterations, 8 threads) — makes GPU cracking expensive as fuck
- **HMAC-SHA256** integrity checks on each chunk — detects if someone messed with your data
- **Secure memory wiping** — passwords zeroed from RAM (no forensics will find them)
- **Session encryption** — quick re-encryption without storing plaintext password

**Note:** This won't protect you from a $5 wrench attack. Physical security is your problem.

## Build

```bash
./build.sh      # Linux/macOS
build.bat       # Windows
```

Requires Go 1.21+

## Languages

English, Русский, Українська

## FAQ

**Q: Can I recover my password if I forget it?**  
A: No. Absolutely not. The math doesn't care about your feelings. Write it down somewhere safe.

**Q: What if I add new files to an encrypted drive?**  
A: They won't be encrypted automatically. This isn't magic. Decrypt → Add files → Re-encrypt.

**Q: Will this work on my potato computer?**  
A: Argon2 needs 1 GB RAM. If your computer has less, it will hang during encryption. Time to upgrade, grandpa.

**Q: Is this actually secure or just security theater?**  
A: Actually secure. Same crypto primitives used by governments. But remember: nothing is 100% unbreakable if someone really wants your data and has infinite time/money.

**Q: Can the NSA crack this?**  
A: Probably not with current technology. But they can torture you for the password, so... ¯\\\_(ツ)_/¯

**Q: Why "UnFuckable"?**  
A: Because your data should be unfuckable. Simple as that.

**Q: What happens if the program crashes during encryption?**  
A: Your original files are deleted only AFTER successful encryption. If it crashes mid-process, your files should still be there. But make backups anyway, don't be stupid.

## Use cases

- **Sensitive documents** you don't want anyone to see
- **Whistleblowing** — transport data safely
- **Privacy** — because it's nobody's business
- **Paranoia** — you can never be too careful
- **Portable secrets** — keep your side hustle private

## Warnings

⚠️ **Don't manually touch encrypted files!** Your data is hidden in chunks. Deleting/moving them = permanent data loss. You've been warned.

⚠️ **Don't forget your password.** It cannot be recovered. Not by me, not by anyone. The laws of mathematics are cruel.

⚠️ **Make backups** before first-time encryption. Shit happens. Don't cry to me if you lose everything.

## License

MIT — do whatever you want with this code. Just don't blame me if something breaks.

---

*Making your data impossible to fuck with.*