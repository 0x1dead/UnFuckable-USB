package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) handleEncrypt() {
	if a.isOperationRunning() {
		return
	}

	form := tview.NewForm()

	var password1, password2 string

	form.AddPasswordField(T("enter_password"), "", 35, '*', func(text string) {
		password1 = text
	})

	form.AddPasswordField(T("confirm_password"), "", 35, '*', func(text string) {
		password2 = text
	})

	form.AddTextView("", T("password_min"), 30, 1, true, false)

	form.AddButton(T("encrypt"), func() {
		if len(password1) < 8 {
			a.showError(T("password_min"))
			return
		}

		if password1 != password2 {
			a.showError(T("password_mismatch"))
			return
		}

		a.pages.RemovePage("encrypt_form")
		a.performEncrypt(password1)
	})

	form.AddButton(T("cancel"), func() {
		a.pages.RemovePage("encrypt_form")
		a.showDeviceMenu()
	})

	form.SetBorder(true).
		SetTitle(" " + T("encrypt") + " ").
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddAndSwitchToPage("encrypt_form", a.centerBox(form, 65, 14), true)
}

func (a *App) handleQuickEncrypt() {
	if a.isOperationRunning() {
		return
	}

	if !a.selected.HasSession {
		a.showError(T("no_session"))
		return
	}

	if AppConfig.ConfirmActions {
		modal := tview.NewModal().
			SetText(T("confirm_encrypt")).
			AddButtons([]string{T("yes"), T("no")}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.pages.RemovePage("confirm_encrypt")
				if buttonIndex == 0 {
					a.performQuickEncrypt()
				}
			})

		a.pages.AddAndSwitchToPage("confirm_encrypt", modal, true)
	} else {
		a.performQuickEncrypt()
	}
}

func (a *App) handleDecrypt() {
	if a.isOperationRunning() {
		return
	}

	// ALWAYS ask for password - session is only for encryption!
	form := tview.NewForm()

	var password string

	form.AddPasswordField(T("enter_password"), "", 40, '*', func(text string) {
		password = text
	})

	form.AddButton(T("decrypt"), func() {
		if len(password) < 8 {
			a.showError(T("password_min"))
			return
		}

		a.pages.RemovePage("decrypt_form")
		a.performDecrypt(password)
	})

	form.AddButton(T("cancel"), func() {
		a.pages.RemovePage("decrypt_form")
		a.showDeviceMenu()
	})

	form.SetBorder(true).
		SetTitle(" " + T("decrypt") + " ").
		SetBorderColor(tcell.ColorBlue)

	a.pages.AddAndSwitchToPage("decrypt_form", a.centerBox(form, 60, 10), true)
}

func (a *App) handleChangePassword() {
	if a.isOperationRunning() {
		return
	}

	form := tview.NewForm()

	var newPass1, newPass2 string

	form.AddPasswordField(T("new_password"), "", 40, '*', func(text string) {
		newPass1 = text
	})

	form.AddPasswordField(T("confirm_password"), "", 40, '*', func(text string) {
		newPass2 = text
	})

	form.AddButton(T("confirm"), func() {
		if len(newPass1) < 8 {
			a.showError(T("password_min"))
			return
		}

		if newPass1 != newPass2 {
			a.showError(T("password_mismatch"))
			return
		}

		oldPass, ok := Sessions.Get(a.selected.DriveID)
		if !ok {
			a.showError(T("no_session"))
			return
		}

		a.pages.RemovePage("change_pass_form")
		a.performChangePassword(oldPass, newPass1)
	})

	form.AddButton(T("cancel"), func() {
		a.pages.RemovePage("change_pass_form")
		a.showDeviceMenu()
	})

	form.SetBorder(true).
		SetTitle(" " + T("change_password") + " ").
		SetBorderColor(tcell.ColorYellow)

	a.pages.AddAndSwitchToPage("change_pass_form", a.centerBox(form, 60, 12), true)
}

func (a *App) handleErase() {
	modal := tview.NewModal().
		SetText("[red]" + T("warning") + "[-]\n\n" + T("confirm_erase")).
		AddButtons([]string{T("yes"), T("no")}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("confirm_erase")
			if buttonIndex == 0 {
				a.performErase()
			}
		})

	a.pages.AddAndSwitchToPage("confirm_erase", modal, true)
}

func (a *App) confirmPanic() {
	modal := tview.NewModal().
		SetText("[red]⚠ " + T("warning") + "[-]\n\n" + T("confirm_panic")).
		AddButtons([]string{T("yes"), T("no")}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("confirm_panic")
			if buttonIndex == 0 {
				a.handlePanicTrigger()
			}
		})

	a.pages.AddAndSwitchToPage("confirm_panic", modal, true)
}

func (a *App) handlePanicTrigger() {
	a.setOperationRunning(true)

	progress := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	progress.SetBorder(true).
		SetTitle(" ⚡ " + T("panic") + " ").
		SetBorderColor(tcell.ColorRed)

	fmt.Fprintf(progress, "\n\n[red]%s[-]\n\n", T("encrypting"))
	fmt.Fprintf(progress, "[yellow]%s[-]", T("please_wait"))

	a.pages.AddAndSwitchToPage("panic_progress", a.centerBox(progress, 50, 10), true)

	go func() {
		Panic.Trigger()
		PanicEncrypt()

		a.app.QueueUpdateDraw(func() {
			a.setOperationRunning(false)
			a.pages.RemovePage("panic_progress")
			a.lastScan = time.Time{}
			a.updateStatusBar(T("done"))
			a.showDeviceList()
		})
	}()
}

func (a *App) performEncrypt(password string) {
	a.setOperationRunning(true)

	progress := a.createProgressView(T("encrypting"))
	a.pages.AddAndSwitchToPage("progress", a.centerBox(progress, 70, 10), true)

	go func() {
		err := EncryptDrive(a.selected.Path, a.selected.DriveID, password, func(current, total int64, stage string) {
			percent := float64(current) / float64(total) * 100
			a.app.QueueUpdateDraw(func() {
				a.updateProgress(progress, stage, int(percent), current, total)
			})
		})

		a.app.QueueUpdateDraw(func() {
			a.setOperationRunning(false)
			a.pages.RemovePage("progress")

			if err != nil {
				a.showError(fmt.Sprintf("%v", err))
			} else {
				a.lastScan = time.Time{}
				a.updateStatusBar(T("success"))
				a.showDeviceList()
			}
		})
	}()
}

func (a *App) performQuickEncrypt() {
	a.setOperationRunning(true)

	progress := a.createProgressView(T("encrypting"))
	a.pages.AddAndSwitchToPage("progress", a.centerBox(progress, 70, 10), true)

	go func() {
		err := QuickEncrypt(a.selected.Path, a.selected.DriveID, func(current, total int64, stage string) {
			percent := float64(current) / float64(total) * 100
			a.app.QueueUpdateDraw(func() {
				a.updateProgress(progress, stage, int(percent), current, total)
			})
		})

		a.app.QueueUpdateDraw(func() {
			a.setOperationRunning(false)
			a.pages.RemovePage("progress")

			if err != nil {
				a.showError(fmt.Sprintf("%v", err))
			} else {
				a.lastScan = time.Time{}
				a.updateStatusBar(T("success"))
				a.showDeviceList()
			}
		})
	}()
}

func (a *App) performDecrypt(password string) {
	a.setOperationRunning(true)

	progress := a.createProgressView(T("decrypting"))
	a.pages.AddAndSwitchToPage("progress", a.centerBox(progress, 70, 10), true)

	go func() {
		err := DecryptDrive(a.selected.Path, a.selected.DriveID, password, func(current, total int64, stage string) {
			percent := float64(current) / float64(total) * 100
			a.app.QueueUpdateDraw(func() {
				a.updateProgress(progress, stage, int(percent), current, total)
			})
		})

		a.app.QueueUpdateDraw(func() {
			a.setOperationRunning(false)
			a.pages.RemovePage("progress")

			if err != nil {
				a.showError(T("wrong_password"))
			} else {
				a.lastScan = time.Time{}
				a.updateStatusBar(T("success"))
				a.showDeviceList()
			}
		})
	}()
}

func (a *App) performChangePassword(oldPassword, newPassword string) {
	a.setOperationRunning(true)

	progress := a.createProgressView(T("processing"))
	a.pages.AddAndSwitchToPage("progress", a.centerBox(progress, 70, 10), true)

	go func() {
		err := ChangePassword(a.selected.Path, a.selected.DriveID, oldPassword, newPassword)

		a.app.QueueUpdateDraw(func() {
			a.setOperationRunning(false)
			a.pages.RemovePage("progress")

			if err != nil {
				a.showError(fmt.Sprintf("%v", err))
			} else {
				a.updateStatusBar(T("password_changed"))
				a.showDeviceMenu()
			}
		})
	}()
}

func (a *App) performErase() {
	err := EraseVault(a.selected.Path, a.selected.DriveID)

	if err != nil {
		a.showError(fmt.Sprintf("%v", err))
	} else {
		a.lastScan = time.Time{}
		a.updateStatusBar(T("success"))
		a.showDeviceList()
	}
}

func (a *App) showVaultInfo() {
	// Need password to read vault info
	password, ok := Sessions.Get(a.selected.DriveID)
	if !ok {
		// Ask for password
		form := tview.NewForm()

		form.AddPasswordField(T("enter_password"), "", 40, '*', func(text string) {
			password = text
		})

		form.AddButton(T("confirm"), func() {
			a.pages.RemovePage("vault_pass_form")
			a.displayVaultInfo(password)
		})

		form.AddButton(T("cancel"), func() {
			a.pages.RemovePage("vault_pass_form")
			a.showDeviceMenu()
		})

		form.SetBorder(true).SetTitle(" " + T("enter_password") + " ")
		a.pages.AddAndSwitchToPage("vault_pass_form", a.centerBox(form, 60, 10), true)
		return
	}

	a.displayVaultInfo(password)
}

func (a *App) displayVaultInfo(password string) {
	manifest, err := GetVaultInfo(a.selected.Path, password)
	if err != nil {
		a.showError(T("wrong_password"))
		return
	}

	info := tview.NewTextView().SetDynamicColors(true)

	fmt.Fprintf(info, "\n [yellow]%s[-]\n\n", T("vault_info"))
	fmt.Fprintf(info, " [grey]%s:[-] %s\n", T("vault_version"), manifest.Version)
	fmt.Fprintf(info, " [grey]%s:[-] %s\n", T("vault_created"), manifest.Created.Format("2006-01-02 15:04"))
	fmt.Fprintf(info, " [grey]%s:[-] %s\n", T("vault_modified"), manifest.Modified.Format("2006-01-02 15:04"))
	fmt.Fprintf(info, " [grey]%s:[-] %d\n", T("vault_files"), manifest.FileCount)
	fmt.Fprintf(info, " [grey]%s:[-] %s\n", T("vault_size"), FormatBytes(uint64(manifest.OriginalSize)))

	if manifest.HasDecoy {
		decoyCount := CountDecoyFiles(a.selected.Path)
		fmt.Fprintf(info, " [grey]%s:[-] %d\n", T("vault_decoys"), decoyCount)
	}

	info.SetBorder(true).SetTitle(" " + T("vault_info") + " ")

	info.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		a.pages.RemovePage("vault_info")
		a.showDeviceMenu()
		return nil
	})

	a.pages.AddAndSwitchToPage("vault_info", a.centerBox(info, 60, 14), true)
}

func (a *App) createProgressView(title string) *tview.TextView {
	progress := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	progress.SetBorder(true).
		SetTitle(" " + title + " ").
		SetBorderColor(tcell.ColorYellow)

	return progress
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
