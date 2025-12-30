package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
)

type Device struct {
	Path        string
	Label       string
	Size        uint64
	Used        uint64
	Free        uint64
	FileSystem  string
	IsEncrypted bool
	DriveID     string
	HasSession  bool
}

type windowsDrive struct {
	DriveLetter string
	BusType     string
	DriveType   int
	Label       string
	SerialNum   string
}

func ScanDevices() ([]Device, error) {
	switch runtime.GOOS {
	case "windows":
		return scanWindows()
	case "linux", "darwin":
		return scanUnix()
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func scanWindows() ([]Device, error) {
	psScript := `
		Get-Partition | Where-Object {$_.DriveLetter} | ForEach-Object {
			$driveLetter = $_.DriveLetter
			$diskNumber = $_.DiskNumber
			$physicalDisk = Get-PhysicalDisk -DeviceNumber $diskNumber -ErrorAction SilentlyContinue
			$logicalDisk = Get-WmiObject Win32_LogicalDisk -Filter "DeviceID='${driveLetter}:'"
			$volume = Get-Volume -DriveLetter $driveLetter -ErrorAction SilentlyContinue
			
			[PSCustomObject]@{
				DriveLetter = $driveLetter
				BusType = $physicalDisk.BusType
				DriveType = $logicalDisk.DriveType
				Label = $volume.FileSystemLabel
				SerialNum = $logicalDisk.VolumeSerialNumber
			}
		} | ConvertTo-Json -Compress
	`

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var drives []windowsDrive
	if err := json.Unmarshal(output, &drives); err != nil {
		var single windowsDrive
		if err := json.Unmarshal(output, &single); err == nil {
			drives = []windowsDrive{single}
		}
	}

	var devices []Device

	for _, d := range drives {
		if d.BusType != "USB" && d.DriveType != 2 {
			continue
		}

		mount := d.DriveLetter + ":\\"
		usage, err := disk.Usage(mount)
		if err != nil {
			continue
		}

		partitions, _ := disk.Partitions(false)
		fs := "Unknown"
		for _, p := range partitions {
			if strings.HasPrefix(p.Mountpoint, d.DriveLetter+":") {
				fs = p.Fstype
				break
			}
		}

		label := d.Label
		if label == "" {
			label = d.DriveLetter + ":"
		}

		driveID := fmt.Sprintf("%s_%s", d.DriveLetter, d.SerialNum)

		dev := Device{
			Path:       mount,
			Label:      label,
			Size:       usage.Total,
			Used:       usage.Used,
			Free:       usage.Free,
			FileSystem: fs,
			DriveID:    driveID,
		}

		dev.IsEncrypted = checkEncrypted(dev.Path)
		dev.HasSession = hasSession(dev.DriveID)

		devices = append(devices, dev)
	}

	return devices, nil
}

func scanUnix() ([]Device, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var devices []Device

	for _, p := range partitions {
		if !isRemovable(p) {
			continue
		}

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		driveID := generateDriveID(p.Mountpoint, p.Device)

		dev := Device{
			Path:       p.Mountpoint,
			Label:      filepath.Base(p.Mountpoint),
			Size:       usage.Total,
			Used:       usage.Used,
			Free:       usage.Free,
			FileSystem: p.Fstype,
			DriveID:    driveID,
		}

		dev.IsEncrypted = checkEncrypted(dev.Path)
		dev.HasSession = hasSession(dev.DriveID)

		devices = append(devices, dev)
	}

	return devices, nil
}

func isRemovable(p disk.PartitionStat) bool {
	switch runtime.GOOS {
	case "linux":
		return strings.HasPrefix(p.Mountpoint, "/media/") ||
			strings.HasPrefix(p.Mountpoint, "/mnt/") ||
			strings.HasPrefix(p.Mountpoint, "/run/media/")
	case "darwin":
		return strings.HasPrefix(p.Mountpoint, "/Volumes/") &&
			p.Mountpoint != "/Volumes/Macintosh HD"
	}
	return false
}

func checkEncrypted(path string) bool {
	manifestPath := filepath.Join(path, ManifestFile)
	_, err := os.Stat(manifestPath)
	return err == nil
}

func hasSession(driveID string) bool {
	_, ok := AppConfig.Sessions[driveID]
	return ok
}

func generateDriveID(mountpoint, device string) string {
	return fmt.Sprintf("%x", HMAC256([]byte(mountpoint+device), []byte("drive_id")))[:16]
}

func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (d *Device) UsagePercent() float64 {
	if d.Size == 0 {
		return 0
	}
	return float64(d.Used) / float64(d.Size) * 100
}

func (d *Device) StatusIcon() string {
	if d.IsEncrypted {
		return "🔒"
	}
	return "🔓"
}

func (d *Device) StatusText() string {
	if d.IsEncrypted {
		return T("encrypted")
	}
	return T("decrypted")
}
