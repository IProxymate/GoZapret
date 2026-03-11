package hosts

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultHostsPath - путь к файлу hosts в Windows
	DefaultHostsPath = `C:\Windows\System32\drivers\etc\hosts`
)

// Service предоставляет функциональность для работы с файлом hosts
type Service struct {
	hostsPath string
}

// NewService создает новый сервис для работы с hosts
func NewService() *Service {
	return &Service{
		hostsPath: DefaultHostsPath,
	}
}

// GetHostsPath возвращает путь к файлу hosts
func (s *Service) GetHostsPath() string {
	return s.hostsPath
}

// Read читает содержимое файла hosts
func (s *Service) Read() (string, error) {
	data, err := os.ReadFile(s.hostsPath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения файла hosts: %w", err)
	}
	return string(data), nil
}

// Write записывает содержимое в файл hosts
func (s *Service) Write(content string) error {
	// Создаем резервную копию перед записью
	if err := s.CreateBackup(); err != nil {
		return fmt.Errorf("ошибка создания резервной копии: %w", err)
	}

	// Записываем новое содержимое
	err := os.WriteFile(s.hostsPath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("ошибка записи файла hosts: %w", err)
	}

	return nil
}

// CreateBackup создает резервную копию файла hosts
func (s *Service) CreateBackup() error {
	// Читаем текущее содержимое
	data, err := os.ReadFile(s.hostsPath)
	if err != nil {
		return err
	}

	// Создаем имя файла резервной копии
	backupDir := filepath.Dir(s.hostsPath)
	backupPath := filepath.Join(backupDir, "hosts.backup")

	// Записываем резервную копию
	return os.WriteFile(backupPath, data, 0644)
}

// Entry представляет запись в файле hosts
type Entry struct {
	IP       string
	Hostname string
	Comment  string
	IsActive bool // false если строка закомментирована
}

// ParseEntries парсит содержимое hosts и возвращает список записей
func (s *Service) ParseEntries(content string) []Entry {
	var entries []Entry
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		entry := s.parseLine(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries
}

// parseLine парсит одну строку файла hosts
func (s *Service) parseLine(line string) *Entry {
	originalLine := line
	isActive := true

	// Проверяем, закомментирована ли строка
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		isActive = false
		// Убираем # и пробелы для парсинга
		line = strings.TrimPrefix(trimmed, "#")
		line = strings.TrimSpace(line)
	}

	// Пропускаем пустые строки
	if line == "" {
		return nil
	}

	// Разделяем на части (IP, hostname, возможный комментарий)
	parts := strings.Fields(line)
	if len(parts) < 2 {
		// Это может быть просто комментарий
		if !isActive && strings.HasPrefix(originalLine, "#") {
			return &Entry{
				Comment:  strings.TrimPrefix(strings.TrimSpace(originalLine), "#"),
				IsActive: false,
			}
		}
		return nil
	}

	ip := parts[0]
	hostname := parts[1]

	// Извлекаем комментарий, если есть
	var comment string
	commentIdx := strings.Index(line, "#")
	if commentIdx > 0 {
		comment = strings.TrimSpace(line[commentIdx+1:])
	}

	return &Entry{
		IP:       ip,
		Hostname: hostname,
		Comment:  comment,
		IsActive: isActive,
	}
}

// AddEntry добавляет новую запись в hosts
func (s *Service) AddEntry(ip, hostname, comment string) error {
	content, err := s.Read()
	if err != nil {
		return err
	}

	// Формируем новую строку
	newLine := fmt.Sprintf("%s\t%s", ip, hostname)
	if comment != "" {
		newLine += fmt.Sprintf("\t# %s", comment)
	}

	// Добавляем в конец файла
	if !strings.HasSuffix(content, "\n") && content != "" {
		content += "\n"
	}
	content += newLine + "\n"

	return s.Write(content)
}

// RemoveEntry удаляет запись из hosts по hostname
func (s *Service) RemoveEntry(hostname string) error {
	content, err := s.Read()
	if err != nil {
		return err
	}

	var newLines []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Пропускаем пустые строки и комментарии
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			newLines = append(newLines, line)
			continue
		}

		// Проверяем, содержит ли строка искомый hostname
		parts := strings.Fields(trimmed)
		if len(parts) >= 2 && strings.EqualFold(parts[1], hostname) {
			// Пропускаем эту строку (удаляем)
			continue
		}

		newLines = append(newLines, line)
	}

	return s.Write(strings.Join(newLines, "\n") + "\n")
}

// ToggleEntry включает/выключает запись (комментирует/раскомментирует)
func (s *Service) ToggleEntry(hostname string) error {
	content, err := s.Read()
	if err != nil {
		return err
	}

	var newLines []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Проверяем закомментированные строки
		if strings.HasPrefix(trimmed, "#") {
			uncommented := strings.TrimPrefix(trimmed, "#")
			uncommented = strings.TrimSpace(uncommented)
			parts := strings.Fields(uncommented)
			if len(parts) >= 2 && strings.EqualFold(parts[1], hostname) {
				// Раскомментируем
				newLines = append(newLines, uncommented)
				continue
			}
		} else {
			// Проверяем активные строки
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 && strings.EqualFold(parts[1], hostname) {
				// Комментируем
				newLines = append(newLines, "# "+line)
				continue
			}
		}

		newLines = append(newLines, line)
	}

	return s.Write(strings.Join(newLines, "\n") + "\n")
}

// ValidateIP проверяет корректность IP адреса (IPv4 и IPv6)
func (s *Service) ValidateIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ValidateHostname проверяет корректность hostname
func (s *Service) ValidateHostname(hostname string) bool {
	if hostname == "" {
		return false
	}

	// Проверяем, что hostname не содержит пробелов
	if strings.Contains(hostname, " ") {
		return false
	}

	// Проверяем допустимые символы
	for _, char := range hostname {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '.' || char == '-' || char == '_') {
			return false
		}
	}

	return true
}
