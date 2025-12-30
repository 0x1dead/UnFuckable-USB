package main

var translations = map[string]map[string]string{
	"en": {
		// App
		"app_name":    "UnFuckable USB",
		"app_tagline": "Making your data impossible to fuck with",

		// Main menu
		"main_menu":   "Main Menu",
		"devices":     "Devices",
		"settings":    "Settings",
		"exclusions":  "Exclusions",
		"sessions":    "Sessions",
		"panic":       "Panic",
		"about":       "About",
		"quit":        "Quit",
		"back":        "Back",
		"select":      "Select",
		"cancel":      "Cancel",
		"confirm":     "Confirm",
		"yes":         "Yes",
		"no":          "No",

		// Devices
		"no_devices":      "No USB devices found",
		"insert_device":   "Insert a USB drive and refresh",
		"refresh":         "Refresh",
		"devices_found":   "devices found",
		"device_info":     "Device Info",
		"device_path":     "Path",
		"device_size":     "Size",
		"device_used":     "Used",
		"device_free":     "Free",
		"device_fs":       "Filesystem",

		// Status
		"encrypted":      "ENCRYPTED",
		"decrypted":      "DECRYPTED",
		"session_active": "Session Active",
		"no_session":     "No Session",

		// Actions
		"encrypt":         "Encrypt",
		"decrypt":         "Decrypt",
		"quick_encrypt":   "Quick Encrypt",
		"change_password": "Change Password",
		"view_info":       "View Info",
		"erase_vault":     "Erase Vault",
		"add_exclusion":   "Add Exclusion",

		// Operations
		"encrypting":  "Encrypting",
		"decrypting":  "Decrypting",
		"archiving":   "Archiving",
		"extracting":  "Extracting",
		"processing":  "Processing",
		"wiping":      "Secure Wiping",
		"done":        "Done",
		"success":     "Success",
		"error":       "Error",
		"warning":     "Warning",
		"canceled":    "Canceled",

		// Password
		"enter_password":   "Enter Password",
		"confirm_password": "Confirm Password",
		"new_password":     "New Password",
		"current_password": "Current Password",
		"password_min":     "Minimum 8 characters",
		"password_mismatch": "Passwords do not match",
		"wrong_password":   "Wrong password",
		"password_changed": "Password changed",

		// Vault info
		"vault_info":     "Vault Info",
		"vault_version":  "Version",
		"vault_created":  "Created",
		"vault_modified": "Modified",
		"vault_files":    "Files",
		"vault_size":     "Original Size",
		"vault_decoys":   "Decoy Files",

		// Settings
		"language":        "Language",
		"theme":           "Theme",
		"auto_lock":       "Auto-lock (minutes)",
		"secure_wipe":     "Secure Wipe",
		"double_encrypt":  "Double Encryption",
		"generate_decoys": "Generate Decoys",
		"decoy_count":     "Decoy Count",
		"confirm_actions": "Confirm Actions",
		"panic_enabled":   "Panic Button",
		"panic_hotkey":    "Panic Hotkey",

		// Confirmations
		"confirm_encrypt": "Encrypt this drive?",
		"confirm_decrypt": "Decrypt this drive?",
		"confirm_erase":   "PERMANENTLY ERASE vault? Cannot be undone!",
		"confirm_panic":   "PANIC: Encrypt ALL decrypted drives NOW?",

		// Exclusions
		"exclusion_pattern": "Pattern",
		"exclusion_add":     "Add Pattern",
		"exclusion_remove":  "Remove",
		"exclusion_help":    "Use * for wildcards, / for directories",

		// Sessions
		"session_drive":    "Drive",
		"session_last":     "Last Used",
		"session_clear":    "Clear",
		"session_clearall": "Clear All",

		// Panic
		"panic_trigger":  "TRIGGER PANIC",
		"panic_status":   "Status",
		"panic_ready":    "Ready",
		"panic_disabled": "Disabled",
		"panic_count":    "Panic Count",
		"panic_last":     "Last Panic",
		"global":         "global",
		"in_app_only":    "in-app only",
		"hotkey_unavailable": "Global hotkey unavailable on this system",

		// About
		"about_version": "Version",
		"about_author":  "Author",
		"about_license": "License",

		// Misc
		"loading":     "Loading...",
		"please_wait": "Please wait...",
		"press_any":   "Press any key",
		"help":        "Help",
		"enabled":     "Enabled",
		"disabled":    "Disabled",
	},

	"ru": {
		// App
		"app_name":    "UnFuckable USB",
		"app_tagline": "Делаем ваши данные невзламываемыми",

		// Main menu
		"main_menu":   "Главное меню",
		"devices":     "Устройства",
		"settings":    "Настройки",
		"exclusions":  "Исключения",
		"sessions":    "Сессии",
		"panic":       "Паника",
		"about":       "О программе",
		"quit":        "Выход",
		"back":        "Назад",
		"select":      "Выбрать",
		"cancel":      "Отмена",
		"confirm":     "Подтвердить",
		"yes":         "Да",
		"no":          "Нет",

		// Devices
		"no_devices":      "USB устройства не найдены",
		"insert_device":   "Вставьте флешку и обновите",
		"refresh":         "Обновить",
		"devices_found":   "устройств найдено",
		"device_info":     "Информация",
		"device_path":     "Путь",
		"device_size":     "Размер",
		"device_used":     "Занято",
		"device_free":     "Свободно",
		"device_fs":       "Файловая система",

		// Status
		"encrypted":      "ЗАШИФРОВАНО",
		"decrypted":      "РАСШИФРОВАНО",
		"session_active": "Сессия активна",
		"no_session":     "Нет сессии",

		// Actions
		"encrypt":         "Зашифровать",
		"decrypt":         "Расшифровать",
		"quick_encrypt":   "Быстрое шифрование",
		"change_password": "Сменить пароль",
		"view_info":       "Информация",
		"erase_vault":     "Удалить хранилище",
		"add_exclusion":   "Добавить исключение",

		// Operations
		"encrypting":  "Шифрование",
		"decrypting":  "Расшифровка",
		"archiving":   "Архивация",
		"extracting":  "Распаковка",
		"processing":  "Обработка",
		"wiping":      "Безопасное удаление",
		"done":        "Готово",
		"success":     "Успешно",
		"error":       "Ошибка",
		"warning":     "Внимание",
		"canceled":    "Отменено",

		// Password
		"enter_password":   "Введите пароль",
		"confirm_password": "Подтвердите пароль",
		"new_password":     "Новый пароль",
		"current_password": "Текущий пароль",
		"password_min":     "Минимум 8 символов",
		"password_mismatch": "Пароли не совпадают",
		"wrong_password":   "Неверный пароль",
		"password_changed": "Пароль изменен",

		// Vault info
		"vault_info":     "Информация о хранилище",
		"vault_version":  "Версия",
		"vault_created":  "Создано",
		"vault_modified": "Изменено",
		"vault_files":    "Файлов",
		"vault_size":     "Исходный размер",
		"vault_decoys":   "Файлов-приманок",

		// Settings
		"language":        "Язык",
		"theme":           "Тема",
		"auto_lock":       "Авто-блокировка (мин)",
		"secure_wipe":     "Безопасное удаление",
		"double_encrypt":  "Двойное шифрование",
		"generate_decoys": "Генерировать приманки",
		"decoy_count":     "Количество приманок",
		"confirm_actions": "Подтверждать действия",
		"panic_enabled":   "Кнопка паники",
		"panic_hotkey":    "Горячая клавиша",

		// Confirmations
		"confirm_encrypt": "Зашифровать этот диск?",
		"confirm_decrypt": "Расшифровать этот диск?",
		"confirm_erase":   "БЕЗВОЗВРАТНО УДАЛИТЬ хранилище?",
		"confirm_panic":   "ПАНИКА: Зашифровать ВСЕ диски СЕЙЧАС?",

		// Exclusions
		"exclusion_pattern": "Паттерн",
		"exclusion_add":     "Добавить паттерн",
		"exclusion_remove":  "Удалить",
		"exclusion_help":    "Используйте * для маски, / для папок",

		// Sessions
		"session_drive":    "Диск",
		"session_last":     "Последнее использование",
		"session_clear":    "Очистить",
		"session_clearall": "Очистить все",

		// Panic
		"panic_trigger":  "ПАНИКА",
		"panic_status":   "Статус",
		"panic_ready":    "Готов",
		"panic_disabled": "Отключено",
		"panic_count":    "Срабатываний",
		"panic_last":     "Последний раз",
		"global":         "глобальный",
		"in_app_only":    "только в приложении",
		"hotkey_unavailable": "Глобальный хоткей недоступен на вашей системе",

		// About
		"about_version": "Версия",
		"about_author":  "Автор",
		"about_license": "Лицензия",

		// Misc
		"loading":     "Загрузка...",
		"please_wait": "Подождите...",
		"press_any":   "Нажмите любую клавишу",
		"help":        "Помощь",
		"enabled":     "Включено",
		"disabled":    "Отключено",
	},

	"uk": {
		// App
		"app_name":    "UnFuckable USB",
		"app_tagline": "Робимо ваші дані невзламуваними",

		// Main menu
		"main_menu":   "Головне меню",
		"devices":     "Пристрої",
		"settings":    "Налаштування",
		"exclusions":  "Виключення",
		"sessions":    "Сесії",
		"panic":       "Паніка",
		"about":       "Про програму",
		"quit":        "Вихід",
		"back":        "Назад",
		"select":      "Вибрати",
		"cancel":      "Скасувати",
		"confirm":     "Підтвердити",
		"yes":         "Так",
		"no":          "Ні",

		// Devices
		"no_devices":      "USB пристрої не знайдено",
		"insert_device":   "Вставте флешку та оновіть",
		"refresh":         "Оновити",
		"devices_found":   "пристроїв знайдено",
		"device_info":     "Інформація",
		"device_path":     "Шлях",
		"device_size":     "Розмір",
		"device_used":     "Зайнято",
		"device_free":     "Вільно",
		"device_fs":       "Файлова система",

		// Status
		"encrypted":      "ЗАШИФРОВАНО",
		"decrypted":      "РОЗШИФРОВАНО",
		"session_active": "Сесія активна",
		"no_session":     "Немає сесії",

		// Actions
		"encrypt":         "Зашифрувати",
		"decrypt":         "Розшифрувати",
		"quick_encrypt":   "Швидке шифрування",
		"change_password": "Змінити пароль",
		"view_info":       "Інформація",
		"erase_vault":     "Видалити сховище",
		"add_exclusion":   "Додати виключення",

		// Operations
		"encrypting":  "Шифрування",
		"decrypting":  "Розшифровка",
		"archiving":   "Архівація",
		"extracting":  "Розпакування",
		"processing":  "Обробка",
		"wiping":      "Безпечне видалення",
		"done":        "Готово",
		"success":     "Успішно",
		"error":       "Помилка",
		"warning":     "Увага",
		"canceled":    "Скасовано",

		// Password
		"enter_password":   "Введіть пароль",
		"confirm_password": "Підтвердіть пароль",
		"new_password":     "Новий пароль",
		"current_password": "Поточний пароль",
		"password_min":     "Мінімум 8 символів",
		"password_mismatch": "Паролі не співпадають",
		"wrong_password":   "Невірний пароль",
		"password_changed": "Пароль змінено",

		// Vault info
		"vault_info":     "Інформація про сховище",
		"vault_version":  "Версія",
		"vault_created":  "Створено",
		"vault_modified": "Змінено",
		"vault_files":    "Файлів",
		"vault_size":     "Початковий розмір",
		"vault_decoys":   "Файлів-приманок",

		// Settings
		"language":        "Мова",
		"theme":           "Тема",
		"auto_lock":       "Авто-блокування (хв)",
		"secure_wipe":     "Безпечне видалення",
		"double_encrypt":  "Подвійне шифрування",
		"generate_decoys": "Генерувати приманки",
		"decoy_count":     "Кількість приманок",
		"confirm_actions": "Підтверджувати дії",
		"panic_enabled":   "Кнопка паніки",
		"panic_hotkey":    "Гаряча клавіша",

		// Confirmations
		"confirm_encrypt": "Зашифрувати цей диск?",
		"confirm_decrypt": "Розшифрувати цей диск?",
		"confirm_erase":   "БЕЗПОВОРОТНО ВИДАЛИТИ сховище?",
		"confirm_panic":   "ПАНІКА: Зашифрувати ВСІ диски ЗАРАЗ?",

		// Exclusions
		"exclusion_pattern": "Патерн",
		"exclusion_add":     "Додати патерн",
		"exclusion_remove":  "Видалити",
		"exclusion_help":    "Використовуйте * для маски, / для папок",

		// Sessions
		"session_drive":    "Диск",
		"session_last":     "Останнє використання",
		"session_clear":    "Очистити",
		"session_clearall": "Очистити все",

		// Panic
		"panic_trigger":  "ПАНІКА",
		"panic_status":   "Статус",
		"panic_ready":    "Готовий",
		"panic_disabled": "Вимкнено",
		"panic_count":    "Спрацювань",
		"panic_last":     "Останній раз",
		"global":         "глобальний",
		"in_app_only":    "тільки в додатку",
		"hotkey_unavailable": "Глобальний хоткей недоступний на вашій системі",

		// About
		"about_version": "Версія",
		"about_author":  "Автор",
		"about_license": "Ліцензія",

		// Misc
		"loading":     "Завантаження...",
		"please_wait": "Зачекайте...",
		"press_any":   "Натисніть будь-яку клавішу",
		"help":        "Допомога",
		"enabled":     "Увімкнено",
		"disabled":    "Вимкнено",
	},
}

// T returns translation for key
func T(key string) string {
	lang := AppConfig.Language

	if trans, ok := translations[lang]; ok {
		if text, ok := trans[key]; ok {
			return text
		}
	}

	// Fallback to English
	if trans, ok := translations["en"]; ok {
		if text, ok := trans[key]; ok {
			return text
		}
	}

	return key
}

// GetLanguages returns available languages
func GetLanguages() []string {
	return []string{"en", "ru", "uk"}
}

// GetLanguageName returns language display name
func GetLanguageName(code string) string {
	switch code {
	case "en":
		return "English"
	case "ru":
		return "Русский"
	case "uk":
		return "Українська"
	}
	return code
}
