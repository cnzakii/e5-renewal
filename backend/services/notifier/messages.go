package notifier

import "fmt"

// FormatTest returns a formatted test notification message.
func FormatTest(lang string) (title, message string) {
	if lang == "en" {
		return "E5 Renewal", "Test notification"
	}
	return "E5 Renewal", "测试通知"
}

// FormatAuthExpiry returns a formatted auth expiry notification.
func FormatAuthExpiry(lang, accountName string, daysLeft int) (title, message string) {
	if lang == "en" {
		if daysLeft < 0 {
			return "Client Secret Expired",
				fmt.Sprintf("Account: %s\nExpired %d days ago", accountName, -daysLeft)
		}
		return "Client Secret Expiring",
			fmt.Sprintf("Account: %s\nExpires in %d days", accountName, daysLeft)
	}
	if daysLeft < 0 {
		return "Client Secret 已过期",
			fmt.Sprintf("账号: %s\n已过期 %d 天", accountName, -daysLeft)
	}
	return "Client Secret 过期提醒",
		fmt.Sprintf("账号: %s\n%d 天后过期", accountName, daysLeft)
}

// FormatTaskAllFailed returns a formatted all-tasks-failed notification.
func FormatTaskAllFailed(lang, accountName string, failCount int) (title, message string) {
	if lang == "en" {
		return "All Tasks Failed",
			fmt.Sprintf("Account: %s\nFailed: %d endpoints", accountName, failCount)
	}
	return "任务全部失败",
		fmt.Sprintf("账号: %s\n失败: %d 个端点", accountName, failCount)
}

// FormatHealthLow returns a formatted health-low notification.
func FormatHealthLow(lang, accountName string, health float64, threshold int) (title, message string) {
	if lang == "en" {
		return "Health Low",
			fmt.Sprintf("Account: %s\nCurrent: %.0f%%\nThreshold: %d%%", accountName, health, threshold)
	}
	return "健康度过低",
		fmt.Sprintf("账号: %s\n当前: %.0f%%\n阈值: %d%%", accountName, health, threshold)
}
