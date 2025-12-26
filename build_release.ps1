# Скрипт сборки релиза GoZapret для Windows
# Использование: .\build_release.ps1 -Version "1.0.1"
#
# Требования:
# - Установлен go-winres: go install github.com/tc-hib/go-winres@latest
# - Обновлена версия в internal/domain/constants.go (DefaultVersion)

param(
    [Parameter(Mandatory=$true)]
    [string]$Version
)

$ErrorActionPreference = "Stop"

# Конфигурация
$AppName = "GoZapret"
$OutputDir = "./release"

Write-Host "=== Сборка релиза $AppName v$Version ===" -ForegroundColor Cyan

# Создаём директорию для релиза
if (Test-Path $OutputDir) {
    try {
        Remove-Item -Recurse -Force $OutputDir -ErrorAction Stop
    } catch {
        Write-Host "Не удалось удалить $OutputDir. Возможно файл запущен." -ForegroundColor Red
        Write-Host "Закройте GoZapret.exe и попробуйте снова." -ForegroundColor Yellow
        exit 1
    }
}
New-Item -ItemType Directory -Path $OutputDir | Out-Null

# Очищаем старые .syso файлы
Write-Host "`nОчистка старых ресурсов..." -ForegroundColor Yellow
Get-ChildItem -Filter "*.syso" -ErrorAction SilentlyContinue | Remove-Item -Force
Write-Host "  Очищено" -ForegroundColor Gray

# Обновляем версию в winres.json
Write-Host "`nОбновление версии в winres/winres.json..." -ForegroundColor Yellow
$winresPath = "./winres/winres.json"
$winresContent = Get-Content $winresPath -Raw
$versionParts = $Version -split '\.'
if ($versionParts.Count -lt 3) {
    $versionParts += @("0") * (3 - $versionParts.Count)
}
$fileVersion = "$($versionParts[0]).$($versionParts[1]).$($versionParts[2]).0"

$winresContent = $winresContent -replace '"file_version": "[^"]*"', "`"file_version`": `"$fileVersion`""
$winresContent = $winresContent -replace '"product_version": "[^"]*"', "`"product_version`": `"$fileVersion`""
$winresContent = $winresContent -replace '"FileVersion": "[^"]*"', "`"FileVersion`": `"$fileVersion`""
$winresContent = $winresContent -replace '"ProductVersion": "[^"]*"', "`"ProductVersion`": `"$fileVersion`""
$winresContent = $winresContent -replace '("identity":\s*\{[^}]*"version":\s*")[^"]*"', "`${1}$fileVersion`""
Set-Content $winresPath $winresContent -NoNewline
Write-Host "  Версия обновлена до $fileVersion" -ForegroundColor Gray

# Генерируем .syso для amd64
Write-Host "`nГенерация ресурсов Windows (go-winres)..." -ForegroundColor Yellow
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go-winres make
if ($LASTEXITCODE -ne 0) {
    throw "Ошибка генерации ресурсов!"
}
Write-Host "  rsrc_windows_amd64.syso создан" -ForegroundColor Gray

# Сборка для Windows amd64
Write-Host "`n[1/1] Сборка для Windows amd64..." -ForegroundColor Green
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"

# Собираем с тегом для отключения генерации ресурсов Fyne
# -s -w убирают отладочную информацию
# -H=windowsgui скрывает консольное окно
go build -ldflags "-s -w -H=windowsgui" -o "$OutputDir/${AppName}.exe" .

if ($LASTEXITCODE -ne 0) {
    throw "Ошибка сборки!"
}

$exeSize = [math]::Round((Get-Item "$OutputDir/${AppName}.exe").Length / 1MB, 2)
Write-Host "  -> ${AppName}.exe создан ($exeSize MB)" -ForegroundColor Gray

# Показываем результат
Write-Host "`n=== Сборка завершена ===" -ForegroundColor Cyan
Write-Host "Файлы для релиза:" -ForegroundColor Yellow
Get-ChildItem $OutputDir -Filter "*.exe" | ForEach-Object {
    $size = [math]::Round($_.Length / 1MB, 2)
    Write-Host "  $OutputDir/$($_.Name) ($size MB)" -ForegroundColor White
}

Write-Host "`n=== Публикация на GitHub ===" -ForegroundColor Cyan
Write-Host @"
1. Перейдите: https://github.com/IProxymate/GoZapret/releases/new
2. Тег: v$Version
3. Название: GoZapret v$Version  
4. Загрузите: $OutputDir/${AppName}.exe
5. Опубликуйте релиз
"@ -ForegroundColor White

Write-Host "`n=== ВАЖНО ===" -ForegroundColor Red
Write-Host "Перед сборкой убедитесь, что обновили версию в:" -ForegroundColor Yellow
Write-Host "  internal/domain/constants.go -> DefaultVersion = `"$Version`"" -ForegroundColor White
