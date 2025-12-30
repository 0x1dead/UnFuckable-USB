package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	app       *tview.Application
	pages     *tview.Pages
	mainFlex  *tview.Flex
	statusBar *tview.TextView
	header    *tview.TextView

	devices     []Device
	selected    *Device
	scanning    bool
	lastScan    time.Time
	scanMu      sync.Mutex
	operationMu sync.Mutex
	isRunning   bool
}

func NewApp() *App {
	LoadConfig()
	Sessions.LoadFromConfig()

	return &App{
		app:   tview.NewApplication(),
		pages: tview.NewPages(),
	}
}

func (a *App) Run() error {
	a.buildUI()

	Panic.SetCallback(func() {
		a.app.QueueUpdateDraw(func() {
			a.handlePanicTrigger()
		})
	})

	if AppConfig.PanicEnabled {
		Panic.Start()
	}

	return a.app.SetRoot(a.mainFlex, true).EnableMouse(true).Run()
}

func (a *App) buildUI() {
	a.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	a.updateHeader()

	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	a.updateStatusBar("")

	a.pages.AddPage("main", a.createMainMenu(), true, true)

	content := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.header, 5, 0, false).
		AddItem(a.pages, 0, 1, true).
		AddItem(a.statusBar, 1, 0, false)

	a.mainFlex = tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(content, UIHeight, 0, true).
			AddItem(nil, 0, 1, false),
			UIWidth, 0, true).
		AddItem(nil, 0, 1, false)

	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyF12 || event.Key() == tcell.KeyCtrlP {
			a.handlePanicTrigger()
			return nil
		}
		return event
	})
}

func (a *App) updateHeader() {
	a.header.Clear()
	fmt.Fprintf(a.header, "\n[green]╔════════════════════════════════════════════════════════════════════════════════════════╗[-]\n")
	fmt.Fprintf(a.header, "[green]║[yellow]       🔐 %s v%s [grey]by %s[-][green]       ║[-]\n", AppName, AppVersion, AppAuthor)
	fmt.Fprintf(a.header, "[green]╚════════════════════════════════════════════════════════════════════════════════════════╝[-]")
}

func (a *App) updateStatusBar(msg string) {
	a.statusBar.Clear()

	panicStatus := "[red]PANIC:OFF[-]"
	if AppConfig.PanicEnabled {
		if IsGlobalHotkeyAvailable() {
			panicStatus = "[green]PANIC:ON (Ctrl+Shift+F12)[-]"
		} else {
			panicStatus = "[yellow]PANIC:ON (F12)[-]"
		}
	}

	sessCount := len(Sessions.GetAll())
	sessInfo := fmt.Sprintf("[blue]Sessions:%d[-]", sessCount)

	if msg != "" {
		fmt.Fprintf(a.statusBar, " %s | %s | %s", panicStatus, sessInfo, msg)
	} else {
		fmt.Fprintf(a.statusBar, " %s | %s", panicStatus, sessInfo)
	}
}

func (a *App) createMainMenu() tview.Primitive {
	list := tview.NewList().
		AddItem(T("devices"), "", '1', func() {
			a.showDeviceList()
		}).
		AddItem(T("settings"), "", '2', func() {
			a.showSettings()
		}).
		AddItem(T("exclusions"), "", '3', func() {
			a.showExclusions()
		}).
		AddItem(T("sessions"), "", '4', func() {
			a.showSessions()
		}).
		AddItem(T("panic"), "", '5', func() {
			a.showPanicMenu()
		}).
		AddItem(T("about"), "", '6', func() {
			a.showAbout()
		}).
		AddItem(T("quit"), "", 'q', func() {
			a.app.Stop()
		})

	list.SetBorder(true).
		SetTitle(" " + T("main_menu") + " ").
		SetBorderColor(tcell.ColorGreen)

	return a.centerBox(list, 50, 14)
}

func (a *App) showDeviceList() {
	a.scanMu.Lock()
	isScanning := a.scanning
	a.scanMu.Unlock()

	if isScanning {
		return
	}

	if time.Since(a.lastScan) < 3*time.Second && len(a.devices) > 0 {
		a.displayDeviceList()
		return
	}

	loader := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	fmt.Fprintf(loader, "\n\n[yellow]%s[-]\n", T("loading"))

	loader.SetBorder(true).SetTitle(" " + T("devices") + " ")
	a.pages.AddAndSwitchToPage("loader", a.centerBox(loader, 50, 8), true)

	go func() {
		a.scanMu.Lock()
		a.scanning = true
		a.scanMu.Unlock()

		devices, err := ScanDevices()

		a.scanMu.Lock()
		a.scanning = false
		a.lastScan = time.Now()
		a.scanMu.Unlock()

		a.app.QueueUpdateDraw(func() {
			a.pages.RemovePage("loader")

			if err != nil {
				a.showError(fmt.Sprintf("Scan error: %v", err))
				return
			}

			a.devices = devices
			a.displayDeviceList()
		})
	}()
}

func (a *App) displayDeviceList() {
	if len(a.devices) == 0 {
		a.showMessage(T("no_devices"), T("insert_device"))
		return
	}

	list := tview.NewList()

	for i, dev := range a.devices {
		idx := i

		icon := "🔓"
		if dev.IsEncrypted {
			icon = "🔒"
		}

		sessionMark := ""
		if dev.HasSession && !dev.IsEncrypted {
			sessionMark = " ✓"
		}

		// Path без \ в конце
		path := strings.TrimSuffix(dev.Path, "\\")
		path = strings.TrimSuffix(path, "/")

		// Не дублируем если label это просто буква диска
		labelPart := ""
		if dev.Label != "" && dev.Label != path && dev.Label != path+":" {
			labelPart = " " + dev.Label
		}

		label := fmt.Sprintf("%s %s%s [%s/%s]%s",
			icon, path, labelPart,
			FormatBytes(dev.Used), FormatBytes(dev.Size),
			sessionMark)

		list.AddItem(label, "", rune('a'+i), func() {
			a.selected = &a.devices[idx]
			a.showDeviceMenu()
		})
	}

	list.AddItem(T("refresh"), "", 'r', func() {
		a.lastScan = time.Time{}
		a.showDeviceList()
	})

	list.AddItem(T("back"), "", 'b', func() {
		a.pages.SwitchToPage("main")
	})

	list.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s (%d) ", T("devices"), len(a.devices))).
		SetBorderColor(tcell.ColorBlue)

	height := len(a.devices) + 6
	if height > 14 {
		height = 14
	}

	a.pages.AddAndSwitchToPage("devices", a.centerBox(list, 60, height), true)
}

func (a *App) showDeviceMenu() {
	if a.selected == nil {
		return
	}

	dev := a.selected

	icon := "🔓"
	status := T("decrypted")
	if dev.IsEncrypted {
		icon = "🔒"
		status = T("encrypted")
	}

	list := tview.NewList()

	// Path без \ в конце
	path := strings.TrimSuffix(dev.Path, "\\")
	path = strings.TrimSuffix(path, "/")

	// Не дублируем если label это просто буква диска
	labelPart := ""
	if dev.Label != "" && dev.Label != path && dev.Label != path+":" {
		labelPart = " " + dev.Label
	}

	// Info в первой строке
	info := fmt.Sprintf("%s %s%s [%s/%s] %s",
		icon, path, labelPart,
		FormatBytes(dev.Used), FormatBytes(dev.Size),
		status)

	list.AddItem(info, "", 0, nil)

	if dev.IsEncrypted {
		list.AddItem(T("decrypt"), "", 'd', func() {
			a.handleDecrypt()
		})

		list.AddItem(T("view_info"), "", 'i', func() {
			a.showVaultInfo()
		})

		list.AddItem(T("erase_vault"), "", 'e', func() {
			a.handleErase()
		})
	} else {
		if dev.HasSession {
			list.AddItem(T("quick_encrypt"), "", 'e', func() {
				a.handleQuickEncrypt()
			})

			list.AddItem(T("change_password"), "", 'c', func() {
				a.handleChangePassword()
			})
		} else {
			list.AddItem(T("encrypt"), "", 'e', func() {
				a.handleEncrypt()
			})
		}
	}

	list.AddItem(T("back"), "", 'b', func() {
		a.displayDeviceList()
	})

	list.SetBorder(true).
		SetTitle(" " + path + " ").
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddAndSwitchToPage("device_menu", a.centerBox(list, 65, 12), true)
}

func (a *App) showSettings() {
	form := tview.NewForm()

	langOptions := []string{"English", "Русский", "Українська"}
	langCodes := []string{"en", "ru", "uk"}
	currentLang := 0
	for i, code := range langCodes {
		if code == AppConfig.Language {
			currentLang = i
			break
		}
	}

	form.AddDropDown(T("language"), langOptions, currentLang, func(option string, index int) {
		AppConfig.Language = langCodes[index]
	})

	form.AddInputField(T("auto_lock"), fmt.Sprintf("%d", AppConfig.AutoLockMinutes), 5, nil, func(text string) {
		var val int
		fmt.Sscanf(text, "%d", &val)
		if val > 0 {
			AppConfig.AutoLockMinutes = val
		}
	})

	form.AddCheckbox(T("secure_wipe"), AppConfig.SecureWipe, func(checked bool) {
		AppConfig.SecureWipe = checked
	})

	form.AddCheckbox(T("double_encrypt"), AppConfig.DoubleEncrypt, func(checked bool) {
		AppConfig.DoubleEncrypt = checked
	})

	form.AddCheckbox(T("generate_decoys"), AppConfig.GenerateDecoys, func(checked bool) {
		AppConfig.GenerateDecoys = checked
	})

	form.AddCheckbox(T("use_chunks"), AppConfig.UseChunks, func(checked bool) {
		AppConfig.UseChunks = checked
	})

	form.AddInputField(T("chunk_size_mb"), fmt.Sprintf("%d", AppConfig.ChunkSizeMB), 5, nil, func(text string) {
		var val int
		fmt.Sscanf(text, "%d", &val)
		if val >= 1 && val <= 50 {
			AppConfig.ChunkSizeMB = val
		}
	})

	form.AddInputField(T("chunk_variance"), fmt.Sprintf("%d", AppConfig.ChunkVariance), 5, nil, func(text string) {
		var val int
		fmt.Sscanf(text, "%d", &val)
		if val >= 0 && val <= 100 {
			AppConfig.ChunkVariance = val
		}
	})

	form.AddCheckbox(T("panic_enabled"), AppConfig.PanicEnabled, func(checked bool) {
		AppConfig.PanicEnabled = checked
		if checked {
			Panic.Start()
		} else {
			Panic.Stop()
		}
	})

	form.AddButton(T("confirm"), func() {
		SaveConfig()
		a.updateStatusBar(T("success"))
		a.buildUI()
		a.pages.SwitchToPage("main")
	})

	form.AddButton(T("cancel"), func() {
		LoadConfig()
		a.pages.SwitchToPage("main")
	})

	form.SetBorder(true).
		SetTitle(" " + T("settings") + " ").
		SetBorderColor(tcell.ColorYellow)

	a.pages.AddAndSwitchToPage("settings", a.centerBox(form, 60, 24), true)
}

func (a *App) showExclusions() {
	form := tview.NewForm()

	exclusions := GetExclusions()

	if len(exclusions) > 0 {
		listText := ""
		for _, excl := range exclusions {
			listText += excl + "\n"
		}
		form.AddTextView(T("exclusions"), listText, 50, 5, true, true)
	}

	form.AddButton(T("exclusion_add"), func() {
		a.showAddExclusion()
	})

	if len(exclusions) > 0 {
		form.AddButton(T("exclusion_remove"), func() {
			a.showRemoveExclusion()
		})
	}

	form.AddButton(T("back"), func() {
		a.pages.SwitchToPage("main")
	})

	form.SetBorder(true).
		SetTitle(" " + T("exclusions") + " ").
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddAndSwitchToPage("exclusions", a.centerBox(form, 60, 14), true)
}

func (a *App) showAddExclusion() {
	form := tview.NewForm()

	var pattern string
	form.AddInputField(T("exclusion_pattern"), "", 40, nil, func(text string) {
		pattern = text
	})

	form.AddTextView("", T("exclusion_help"), 50, 2, true, false)

	form.AddButton(T("confirm"), func() {
		if pattern != "" {
			AddExclusion(pattern)
		}
		a.showExclusions()
	})

	form.AddButton(T("cancel"), func() {
		a.showExclusions()
	})

	form.SetBorder(true).SetTitle(" " + T("exclusion_add") + " ")
	a.pages.AddAndSwitchToPage("add_exclusion", a.centerBox(form, 60, 12), true)
}

func (a *App) showRemoveExclusion() {
	form := tview.NewForm()

	exclusions := GetExclusions()
	if len(exclusions) == 0 {
		a.showExclusions()
		return
	}

	form.AddDropDown(T("exclusion_pattern"), exclusions, 0, nil)

	form.AddButton(T("confirm"), func() {
		_, pattern := form.GetFormItem(0).(*tview.DropDown).GetCurrentOption()
		RemoveExclusion(pattern)
		a.showExclusions()
	})

	form.AddButton(T("cancel"), func() {
		a.showExclusions()
	})

	form.SetBorder(true).SetTitle(" " + T("exclusion_remove") + " ")
	a.pages.AddAndSwitchToPage("remove_exclusion", a.centerBox(form, 60, 10), true)
}

func (a *App) showSessions() {
	form := tview.NewForm()

	sessions := Sessions.GetSessionsInfo()

	if len(sessions) > 0 {
		listText := ""
		for _, sess := range sessions {
			listText += fmt.Sprintf("%s - %s\n", sess.DriveID[:8], sess.LastUsed.Format("2006-01-02 15:04"))
		}
		form.AddTextView(T("sessions"), listText, 50, 5, true, true)

		form.AddButton(T("session_clearall"), func() {
			Sessions.ClearAll()
			a.showSessions()
		})
	} else {
		form.AddTextView("", T("no_session"), 50, 2, true, false)
	}

	form.AddButton(T("back"), func() {
		a.pages.SwitchToPage("main")
	})

	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s (%d) ", T("sessions"), len(sessions))).
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddAndSwitchToPage("sessions", a.centerBox(form, 60, 12), true)
}

func (a *App) showPanicMenu() {
	form := tview.NewForm()

	status := T("panic_disabled")
	if AppConfig.PanicEnabled {
		status = T("panic_ready")
	}

	count, lastTime := Panic.GetPanicStats()
	lastStr := "-"
	if !lastTime.IsZero() {
		lastStr = lastTime.Format("2006-01-02 15:04:05")
	}

	info := fmt.Sprintf("%s: %s\n%s: %d\n%s: %s\n%s: %s",
		T("panic_status"), status,
		T("panic_count"), count,
		T("panic_last"), lastStr,
		T("panic_hotkey"), GetHotkeyStatus())

	form.AddTextView("", info, 55, 5, true, false)

	if !IsGlobalHotkeyAvailable() {
		form.AddTextView("", "[yellow]"+T("hotkey_unavailable")+"[-]", 55, 1, true, false)
	}

	form.AddButton(T("panic_trigger"), func() {
		a.confirmPanic()
	})

	toggleText := T("enabled")
	if AppConfig.PanicEnabled {
		toggleText = T("disabled")
	}
	form.AddButton(toggleText, func() {
		AppConfig.PanicEnabled = !AppConfig.PanicEnabled
		SaveConfig()
		if AppConfig.PanicEnabled {
			Panic.Start()
		} else {
			Panic.Stop()
		}
		a.updateStatusBar("")
		a.showPanicMenu()
	})

	form.AddButton(T("back"), func() {
		a.pages.SwitchToPage("main")
	})

	form.SetBorder(true).
		SetTitle(" " + T("panic") + " ").
		SetBorderColor(tcell.ColorRed)

	a.pages.AddAndSwitchToPage("panic_menu", a.centerBox(form, 65, 16), true)
}

func (a *App) showAbout() {
	form := tview.NewForm()

	info := fmt.Sprintf("%s\n\n%s\n\n%s: %s\n%s: %s\n%s: MIT\n\n© %s",
		AppName,
		AppTagline,
		T("about_version"), AppVersion,
		T("about_author"), AppAuthor,
		T("about_license"),
		AppYear)

	form.AddTextView("", info, 55, 10, true, false)

	form.AddButton(T("back"), func() {
		a.pages.SwitchToPage("main")
	})

	form.SetBorder(true).
		SetTitle(" " + T("about") + " ").
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddAndSwitchToPage("about", a.centerBox(form, 65, 16), true)
}

func (a *App) showMessage(title, message string) {
	form := tview.NewForm()

	form.AddTextView("", message, 50, 3, true, false)

	form.AddButton("OK", func() {
		a.pages.SwitchToPage("main")
	})

	form.SetBorder(true).SetTitle(" " + title + " ")
	a.pages.AddAndSwitchToPage("message", a.centerBox(form, 55, 10), true)
}

func (a *App) showError(message string) {
	form := tview.NewForm()

	form.AddTextView("", "[red]"+T("error")+"[-]\n\n"+message, 50, 4, true, false)

	form.AddButton("OK", func() {
		a.pages.RemovePage("error")
	})

	form.SetBorder(true).
		SetTitle(" " + T("error") + " ").
		SetBorderColor(tcell.ColorRed)

	a.pages.AddAndSwitchToPage("error", a.centerBox(form, 55, 10), true)
}

func (a *App) centerBox(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 0, true).
			AddItem(nil, 0, 1, false),
			width, 0, true).
		AddItem(nil, 0, 1, false)
}

func (a *App) isOperationRunning() bool {
	a.operationMu.Lock()
	defer a.operationMu.Unlock()
	return a.isRunning
}

func (a *App) setOperationRunning(running bool) {
	a.operationMu.Lock()
	defer a.operationMu.Unlock()
	a.isRunning = running
}
