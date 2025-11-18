package services

import (
	"golang.org/x/sys/windows"
)

// AdminChecker проверяет права администратора
type AdminChecker struct{}

// NewAdminChecker создает новый проверяльщик прав администратора
func NewAdminChecker() *AdminChecker {
	return &AdminChecker{}
}

// IsAdmin проверяет, запущено ли приложение с правами администратора
func (a *AdminChecker) IsAdmin() bool {
	var sid *windows.SID

	// Получаем SID группы администраторов
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	// Проверяем членство в группе администраторов
	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}

	return member
}
