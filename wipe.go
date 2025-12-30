package main

import (
	"crypto/rand"
	"os"
)

// SecureDelete overwrites file with random data before deletion
func SecureDelete(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(path)
	}

	size := info.Size()

	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return os.Remove(path)
	}

	// Multiple passes of random data
	for pass := 0; pass < WipePasses; pass++ {
		f.Seek(0, 0)

		// Write in chunks
		chunkSize := int64(1024 * 1024) // 1 MB
		buf := make([]byte, chunkSize)

		remaining := size
		for remaining > 0 {
			writeSize := chunkSize
			if remaining < chunkSize {
				writeSize = remaining
			}

			rand.Read(buf[:writeSize])
			f.Write(buf[:writeSize])
			remaining -= writeSize
		}

		f.Sync()
	}

	// Final pass with zeros
	f.Seek(0, 0)
	zeros := make([]byte, 1024*1024)
	remaining := size
	for remaining > 0 {
		writeSize := int64(len(zeros))
		if remaining < writeSize {
			writeSize = remaining
		}
		f.Write(zeros[:writeSize])
		remaining -= writeSize
	}

	f.Sync()
	f.Close()

	// Rename to random name before deletion
	newPath := path + "." + RandomHex(8)
	os.Rename(path, newPath)

	return os.Remove(newPath)
}

// SecureDeleteDir recursively wipes directory
func SecureDeleteDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, e := range entries {
		entryPath := path + string(os.PathSeparator) + e.Name()
		if e.IsDir() {
			SecureDeleteDir(entryPath)
		} else {
			SecureDelete(entryPath)
		}
	}

	return os.Remove(path)
}

// WipeBuffer securely zeroes buffer
func WipeBuffer(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

// WipeString attempts to wipe string from memory
func WipeString(s *string) {
	if s == nil || *s == "" {
		return
	}

	// Can't really wipe Go strings, but we can try
	b := []byte(*s)
	WipeBuffer(b)
	*s = ""
}
