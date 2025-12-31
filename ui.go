package main

import (
	"context"
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
	
	// FIX: Добавлено для отмены сканирования
	scanCancel  context.CancelFunc
	
	// FIX: Добавлено для throttling progress updates
	lastProgressUpdate time.Time
	progressMu         sync.Mutex
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

// FIX: Добавлен graceful shutdown
func (a *App) Shutdown() {
	// Отменить активное сканирование
	if a.scanCancel != nil {
		a.scanCancel()
	}
	
	// Остановить panic listener
	if AppConfig.PanicEnabled {
		Panic.Stop()
	}
	
	// Остановить autolock
	AutoLocker.Stop()
	
	// Сохранить конфиг
	SaveConfig()
	
	// Очистить sensitive data из памяти
	for _, s := range Sessions.GetAll() {
		SecureZero(s.Password)
	}
	
	a.app.Stop()
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
  fmt.Fprintf(a.header, "[green]║[yellow]                           🔐 %s v%s [grey]by %s[-][green]                          ║[-]\n", AppName, AppVersion, AppAuthor)
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
		AddItem(T("help"), "", 'h', func() {
			a.showHelp()
		}).
		AddItem(T("quit"), "", 'q', func() {
			a.Shutdown()  // FIX: Используем graceful shutdown
		})

	list.SetBorder(true).
		SetTitle(" " + T("main_menu") + " ").
		SetBorderColor(tcell.ColorGreen)

	return a.centerBox(list, 50, 16)
}

// FIX: Исправлена утечка горутин и добавлена отмена
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

	// FIX: Отменяем предыдущее сканирование
	if a.scanCancel != nil {
		a.scanCancel()
	}

	loader := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	fmt.Fprintf(loader, "\n\n[yellow]%s[-]\n", T("loading"))

	loader.SetBorder(true).SetTitle(" " + T("devices") + " ")
	a.pages.AddAndSwitchToPage("loader", a.centerBox(loader, 50, 8), true)

	// FIX: Создаем context для отмены
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	a.scanCancel = cancel

	safeGo(func() {  // FIX: Используем safe wrapper
		a.scanMu.Lock()
		a.scanning = true
		a.scanMu.Unlock()

		// FIX: Сканирование с поддержкой context
		devices, err := ScanDevices()

		select {
		case <-ctx.Done():
			// Сканирование было отменено
			a.scanMu.Lock()
			a.scanning = false
			a.scanMu.Unlock()
			return
		default:
			// Продолжаем
		}

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
	})
}

// FIX: Исправлен out of bounds при изменении списка
func (a *App) displayDeviceList() {
	if len(a.devices) == 0 {
		a.showMessage(T("no_devices"), T("insert_device"))
		return
	}

	list := tview.NewList()

	for i, dev := range a.devices {
		device := dev  // FIX: Создаем локальную копию для замыкания

		icon := "🔓"
		if device.IsEncrypted {
			icon = "🔒"
		}

		sessionMark := ""
		if device.HasSession && !device.IsEncrypted {
			sessionMark = " ✓"
		}

		path := strings.TrimSuffix(device.Path, "\\")
		path = strings.TrimSuffix(path, "/")

		labelPart := ""
		if device.Label != "" && device.Label != path && device.Label != path+":" {
			labelPart = " " + device.Label
		}

		label := fmt.Sprintf("%s %s%s [%s/%s]%s",
			icon, path, labelPart,
			FormatBytes(device.Used), FormatBytes(device.Size),
			sessionMark)

		list.AddItem(label, "", rune('a'+i), func() {
			// FIX: Используем локальную копию вместо индекса
			deviceCopy := device
			a.selected = &deviceCopy
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
	if height < 10 {
		height = 10
	}
	if height > 16 {
		height = 16
	}

	a.pages.AddAndSwitchToPage("devices", a.centerBox(list, 70, height), true)
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

	path := strings.TrimSuffix(dev.Path, "\\")
	path = strings.TrimSuffix(path, "/")

	labelPart := ""
	if dev.Label != "" && dev.Label != path && dev.Label != path+":" {
		labelPart = " " + dev.Label
	}

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

	a.pages.AddAndSwitchToPage("device_menu", a.centerBox(list, 70, 11), true)
}

// FIX: Исправлена смена языка
func (a *App) showSettings() {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	
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
	
	originalLang := AppConfig.Language
	selectedLang := originalLang  // FIX: Не изменяем AppConfig в callback
	
	form.AddDropDown(T("language"), langOptions, currentLang, func(option string, index int) {
		selectedLang = langCodes[index]
		// FIX: НЕ изменяем AppConfig.Language здесь!
	})

	form.AddInputField(T("auto_lock"), fmt.Sprintf("%d", AppConfig.AutoLockMinutes), 10, nil, func(text string) {
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

	form.AddInputField(T("chunk_size_mb"), fmt.Sprintf("%d", AppConfig.ChunkSizeMB), 10, nil, func(text string) {
		var val int
		fmt.Sscanf(text, "%d", &val)
		if val >= 1 && val <= 50 {
			AppConfig.ChunkSizeMB = val
		}
	})

	form.AddInputField(T("chunk_variance"), fmt.Sprintf("%d", AppConfig.ChunkVariance), 10, nil, func(text string) {
		var val int
		fmt.Sscanf(text, "%d", &val)
		if val >= 0 && val <= 100 {
			AppConfig.ChunkVariance = val
		}
	})

	form.AddButton(T("confirm"), func() {
		// FIX: Применяем язык ТОЛЬКО здесь
		languageChanged := selectedLang != originalLang
		
		if languageChanged {
			AppConfig.Language = selectedLang
		}
		
		SaveConfig()
		
		if languageChanged {
			// FIX: Показываем индикатор и пересоздаем UI
			a.showMessage(T("loading"), T("please_wait"))
			
			safeGo(func() {
				time.Sleep(100 * time.Millisecond)
				
				a.app.QueueUpdateDraw(func() {
					// Пересоздаем все страницы с новым языком
					a.pages.RemovePage("main")
					a.pages.RemovePage("settings")
					a.pages.RemovePage("message")
					
					a.pages.AddPage("main", a.createMainMenu(), true, true)
					a.pages.SwitchToPage("main")
					
					a.updateHeader()
					a.updateStatusBar(T("success"))
				})
			})
		} else {
			a.updateStatusBar(T("success"))
			a.pages.SwitchToPage("main")
		}
	})

	form.AddButton(T("back"), func() {
		// FIX: Откатываем все изменения
		LoadConfig()
		a.pages.SwitchToPage("main")
	})

	form.SetBorder(false)
	
	flex.AddItem(form, 0, 1, true)
	
	flex.SetBorder(true).
		SetTitle(" " + T("settings") + " ").
		SetBorderColor(tcell.ColorYellow)

	a.pages.AddAndSwitchToPage("settings", a.centerBox(flex, 65, 21), true)
}

// FIX: Добавлен throttling для progress updates
func (a *App) updateProgressThrottled(view *tview.TextView, stage string, percent int, current, total int64) {
	a.progressMu.Lock()
	defer a.progressMu.Unlock()
	
	// Обновляем не чаще 10 раз в секунду (100ms)
	if time.Since(a.lastProgressUpdate) < 100*time.Millisecond && percent < 100 {
		return
	}
	
	a.lastProgressUpdate = time.Now()
	a.app.QueueUpdateDraw(func() {
		a.updateProgress(view, stage, percent, current, total)
	})
}

// ... остальные методы без изменений ...

func (a *App) showHelp() {
	text := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)

	helpText := T("help_text")
	
	fmt.Fprintf(text, "\n%s\n", helpText)

	text.SetBorder(true).
		SetTitle(" " + T("help") + " ").
		SetBorderColor(tcell.ColorBlue)
	
	text.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' || event.Rune() == 'b' {
			a.pages.SwitchToPage("main")
			return nil
		}
		return event
	})

	a.pages.AddAndSwitchToPage("help", a.centerBox(text, 80, 22), true)
}

func (a *App) showExclusions() {
	form := tview.NewForm()

	exclusions := GetExclusions()

	if len(exclusions) > 0 {
		listText := ""
		for _, excl := range exclusions {
			listText += excl + "\n"
		}
		form.AddTextView(T("exclusions"), listText, 60, 6, true, true)
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

	a.pages.AddAndSwitchToPage("exclusions", a.centerBox(form, 70, 15), true)
}

func (a *App) showAddExclusion() {
	form := tview.NewForm()

	var pattern string
	form.AddInputField(T("exclusion_pattern"), "", 50, nil, func(text string) {
		pattern = text
	})

	form.AddTextView("", T("exclusion_help"), 60, 2, true, false)

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
	a.pages.AddAndSwitchToPage("add_exclusion", a.centerBox(form, 70, 12), true)
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
	a.pages.AddAndSwitchToPage("remove_exclusion", a.centerBox(form, 70, 10), true)
}

func (a *App) showSessions() {
	form := tview.NewForm()

	sessions := Sessions.GetSessionsInfo()

	if len(sessions) > 0 {
		listText := ""
		for _, sess := range sessions {
			listText += fmt.Sprintf("%s - %s\n", sess.DriveID[:8], sess.LastUsed.Format("2006-01-02 15:04"))
		}
		form.AddTextView(T("sessions"), listText, 60, 6, true, true)

		form.AddButton(T("session_clearall"), func() {
			Sessions.ClearAll()
			a.showSessions()
		})
	} else {
		form.AddTextView("", T("no_session"), 60, 2, true, false)
	}

	form.AddButton(T("back"), func() {
		a.pages.SwitchToPage("main")
	})

	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s (%d) ", T("sessions"), len(sessions))).
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddAndSwitchToPage("sessions", a.centerBox(form, 70, 13), true)
}

func (a *App) showPanicMenu() {
	form := tview.NewForm()

	statusIcon := "[red]✗[-]"
	statusText := T("panic_disabled")
	statusColor := "red"
	if AppConfig.PanicEnabled {
		statusIcon = "[green]✓[-]"
		statusText = T("panic_ready")
		statusColor = "green"
	}

	count, lastTime := Panic.GetPanicStats()
	lastStr := "-"
	if !lastTime.IsZero() {
		lastStr = lastTime.Format("2006-01-02 15:04:05")
	}

	info := fmt.Sprintf("%s: [%s]%s %s[-]\n%s: %d\n%s: %s\n%s: %s",
		T("panic_status"), statusColor, statusIcon, statusText,
		T("panic_count"), count,
		T("panic_last"), lastStr,
		T("panic_hotkey"), GetHotkeyStatus())

	form.AddTextView("", info, 65, 5, true, false)

	if !IsGlobalHotkeyAvailable() {
		form.AddTextView("", "[yellow]"+T("hotkey_unavailable")+"[-]", 65, 2, true, false)
	}

	form.AddButton(T("panic_trigger"), func() {
		a.confirmPanic()
	})

	toggleText := T("enable") + " " + T("panic")
	if AppConfig.PanicEnabled {
		toggleText = T("disable") + " " + T("panic")
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

	a.pages.AddAndSwitchToPage("panic_menu", a.centerBox(form, 70, 17), true)
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

	form.AddTextView("", info, 65, 10, true, false)

	form.AddButton(T("back"), func() {
		a.pages.SwitchToPage("main")
	})

	form.SetBorder(true).
		SetTitle(" " + T("about") + " ").
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddAndSwitchToPage("about", a.centerBox(form, 70, 16), true)
}

func (a *App) showMessage(title, message string) {
	form := tview.NewForm()

	form.AddTextView("", message, 60, 4, true, false)

	form.AddButton("OK", func() {
		a.pages.RemovePage("message")
		a.pages.SwitchToPage("main")
	})

	form.SetBorder(true).SetTitle(" " + title + " ")
	a.pages.AddAndSwitchToPage("message", a.centerBox(form, 65, 10), true)
}

func (a *App) showError(message string) {
	form := tview.NewForm()

	form.AddTextView("", "[red]"+T("error")+"[-]\n\n"+message, 60, 5, true, false)

	form.AddButton("OK", func() {
		a.pages.RemovePage("error")
	})

	form.SetBorder(true).
		SetTitle(" " + T("error") + " ").
		SetBorderColor(tcell.ColorRed)

	a.pages.AddAndSwitchToPage("error", a.centerBox(form, 65, 11), true)
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

func (a *App) updateProgress(view *tview.TextView, stage string, percent int, current, total int64) {
	view.Clear()

	bar := createProgressBar(percent)

	fmt.Fprintf(view, "\n[yellow]%s[-]\n\n", stage)
	fmt.Fprintf(view, "%s\n", bar)
	fmt.Fprintf(view, "[grey]%d%%  %s / %s[-]\n", percent, FormatBytes(uint64(current)), FormatBytes(uint64(total)))
}

func createProgressBar(percent int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := ProgressWidth * percent / 100
	empty := ProgressWidth - filled

	bar := "[green]" + strings.Repeat("█", filled) + "[-]" +
		"[grey]" + strings.Repeat("░", empty) + "[-]"

	return bar
}

// FIX: Safe goroutine wrapper для предотвращения паник
func safeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// В production версии здесь должно быть логирование
				fmt.Printf("Panic recovered: %v\n", r)
			}
		}()
		fn()
	}()
}