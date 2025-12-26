<div align="center">

# GoZapret

![GoZapret Icon](assets/icon256.png)

Графический интерфейс для [zapret-discord-youtube](https://github.com/Flowseal/zapret-discord-youtube)

[![GitHub release](https://img.shields.io/github/v/release/IProxymate/GoZapret)](https://github.com/IProxymate/GoZapret/releases)

</div>

## Возможности

### Управление стратегиями
- 🚀 Запуск и остановка стратегий в один клик
- 📋 Автоматическое обнаружение .bat файлов стратегий
- 💾 Сохранение последней использованной стратегии
- ⚡ Автозапуск при старте Windows

### Настройки
- 🎮 **Game Filter** — фильтрация UDP трафика для игр
- 🌐 **Режимы IPset** — выбор режима работы (any/none/loaded)
- 📝 **Списки доменов** — добавление своих доменов для включения/исключения
- 🔧 **Пользовательские подсети** — настройка IPset

### Инструменты
- 🔍 **Диагностика системы** — проверка прав администратора, драйвера WinDivert, сетевого подключения, конфликтующих программ
- 🌐 **Проверка домена** — тестирование доступности сайтов
- 📊 **Мониторинг** — отслеживание состояния приложения
- 🧹 **Очистка кэша Discord**

### Обновления
- 🔄 Автоматическое обновление приложения из GitHub
- 📦 Обновление ресурсов zapret
- 📋 Обновление списка IPset

<div align="center">
<img width="802" alt="GoZapret Screenshot" src="https://github.com/user-attachments/assets/e44d575c-ce37-49e4-8427-81e21e18311c" />
</div>

## Установка

1. Скачайте и распакуйте [zapret-discord-youtube](https://github.com/Flowseal/zapret-discord-youtube/releases)
2. Скачайте [GoZapret.exe](https://github.com/IProxymate/GoZapret/releases/latest)
3. Запустите GoZapret и укажите путь к папке zapret

## Использование

1. Выберите стратегию из списка
2. При необходимости включите Game Filter или настройте режим IPset
3. Нажмите **Запустить**
4. Готово!

## Требования

- Windows 10/11
- Права администратора

## Сборка из исходников

```powershell
git clone https://github.com/IProxymate/GoZapret
cd GoZapret

# Установка зависимостей
go mod download
go install github.com/tc-hib/go-winres@latest

# Генерация ресурсов Windows (иконка, манифест администратора)
go-winres make

# Сборка
go build -ldflags "-s -w -H=windowsgui" .
```

Или используйте скрипт для сборки релиза:
```powershell
.\build_release.ps1 -Version "1.0.0"
```

## Благодарности

- [zapret](https://github.com/bol-van/zapret) — bol-van
- [zapret-discord-youtube](https://github.com/Flowseal/zapret-discord-youtube) — Flowseal
