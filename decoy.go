package main

import (
	"crypto/rand"
	"math/big"
	"os"
	"path/filepath"
)

var decoyPrefixes = []string{
	"sys", "tmp", "cache", "data", "log", "db", "idx", "bak",
	"cfg", "inf", "dat", "bin", "lib", "obj", "res", "pkg",
	"mod", "ref", "lnk", "ptr", "buf", "stk", "heap", "mem",
	"reg", "vol", "sec", "key", "sig", "crt", "pub", "prv",
	"enc", "dec", "hash", "chk", "sum", "crc", "md5", "sha",
	"aes", "rsa", "dsa", "ecdsa", "hmac", "kdf", "pbkdf",
	"init", "boot", "kern", "drv", "svc", "proc", "thrd",
	"sock", "pipe", "fifo", "shm", "sem", "mtx", "evt",
}

var decoyExtensions = []string{
	"", "dat", "bin", "sys", "tmp", "bak", "old", "new",
	"0", "1", "2", "db", "idx", "log", "cache",
}

func generateDecoyFileNames(count int) []string {
	names := make([]string, count)

	for i := 0; i < count; i++ {
		prefix := decoyPrefixes[randomInt(len(decoyPrefixes))]
		hex := RandomHex(randomInt(8) + 4)
		ext := decoyExtensions[randomInt(len(decoyExtensions))]

		if ext == "" {
			names[i] = prefix + "_" + hex
		} else {
			names[i] = prefix + "_" + hex + "." + ext
		}
	}

	return names
}

func generateDecoyData() []byte {
	size := MinDecoySize + randomInt(MaxDecoySize-MinDecoySize)
	data := make([]byte, size)
	rand.Read(data)
	return data
}

func randomInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}

func removeDecoyFiles(drivePath string) {
	entries, _ := os.ReadDir(drivePath)

	for _, e := range entries {
		name := e.Name()

		// Skip manifest
		if name == ManifestFile {
			continue
		}

		// Remove all hidden files (they are decoys or vault files)
		if len(name) > 0 && name[0] == '.' {
			path := filepath.Join(drivePath, name)
			if AppConfig.SecureWipe {
				SecureDelete(path)
			} else {
				os.Remove(path)
			}
		}
	}
}

func CountDecoyFiles(drivePath string) int {
	count := 0
	entries, _ := os.ReadDir(drivePath)

	for _, e := range entries {
		name := e.Name()
		if len(name) > 0 && name[0] == '.' && name != ManifestFile {
			count++
		}
	}

	return count
}
