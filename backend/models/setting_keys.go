package models

// Setting table key constants.
// All keys used in the setting table are defined here.
//
// Value storage rules:
//   - Single value (string/bool/number): store directly as string
//     e.g. JWTSecret → "a1b2c3..."
//   - Multiple related fields: JSON serialization
//     e.g. Notification → {"url":"...","on_credential_expiry":true,...}

const (
	// SettingKeyLoginKey stores the login key as a plain hex string.
	SettingKeyLoginKey = "login_key"

	// SettingKeyNotification stores notification config as JSON.
	// Value format: {"url":"...","on_credential_expiry":bool,"expiry_days_before":int,
	//   "on_task_all_failed":bool,"on_health_low":bool,"health_threshold":int}
	SettingKeyNotification = "notification"
)
