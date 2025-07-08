package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qzeleza/macbat/internal/background"
	"github.com/qzeleza/macbat/internal/config"
	"github.com/qzeleza/macbat/internal/env"
	"github.com/qzeleza/macbat/internal/logger"
	"github.com/qzeleza/macbat/internal/monitor"
	"github.com/qzeleza/macbat/internal/paths"
	"github.com/qzeleza/macbat/internal/utils"
)

type Commands struct {
	log *logger.Logger
	cfg *config.Config
}

func NewCommands(log *logger.Logger, cfg *config.Config) *Commands {
	return &Commands{
		log: log,
		cfg: cfg,
	}
}

// Install устанавливает приложение и регистрирует его как агент launchd.
//
// @param log *logger.Logger - логгер
// @return *appConfig.Config - конфигурация приложения
// @return error - ошибка, если не удалось установить приложение
func (c *Commands) Install() error {

	// 1. Определяем пути
	binPath := paths.BinaryPath()
	binDir := paths.BinaryPath()
	// currentBin, err := os.Executable()
	// if err != nil {
	// 	mess := fmt.Sprintf("не удалось определить путь к текущему исполняемому файлу: %v", err)
	// 	log.Error(mess)
	// 	return fmt.Errorf("%s", mess)
	// }

	// log.Debug(fmt.Sprintf("Целевой путь бинарника: %s", binPath))
	// log.Debug(fmt.Sprintf("Текущий путь бинарника: %s", currentBin))

	// Создаем директорию для логов
	if err := c.createLogDirectory(); err != nil {
		return err
	}

	// Добавляем директорию в PATH
	c.addPathToEnvironment(binDir)

	// Создаем plist файл для агента
	if err := c.createPlistFile(binPath); err != nil {
		return fmt.Errorf("не удалось создать plist: %w", err)
	}

	// Отключаем и выгружаем агента
	if err := monitor.UnloadAndDisableAgent(c.log); err != nil {
		c.log.Error(fmt.Sprintf("Ошибка отключения агента: %v", err))
	}
	// Включаем и загружаем агента
	if err := monitor.LoadAndEnableAgent(c.log); err != nil {
		c.log.Error(fmt.Sprintf("Ошибка включения агента: %v", err))
	}

	return nil
}

// createLogDirectory создает директорию для логов, если она не существует.
// @param log *logger.Logger - логгер для вывода отладочной информации.
// @return error - ошибка, если не удалось создать директорию
func (c *Commands) createLogDirectory() error {
	logDir := filepath.Dir(paths.LogPath())
	c.log.Debug(fmt.Sprintf("Создание директории для логов: %s", logDir))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		c.log.Error(fmt.Sprintf("Ошибка создания директории для логов: %v", err))
		return fmt.Errorf("не удалось создать директорию для логов: %v", err)
	}
	c.log.Debug("Директория для логов успешно создана")
	return nil
}

// addPathToEnvironment добавляет указанную директорию в переменную PATH.
//
// Функция сначала пытается добавить директорию в системную переменную PATH,
// используя внутrenний метод AddToPath. Если добавление не удается,
// это регистрируется как предупреждение, и пользователю предлагается
// выполнить команду вручную для добавления директории в PATH.
// После успешного добавления функция пытается обновить текущую сессию оболочки
// с помощью внутреннего метода UpdateShell. Если обновление не удается,
// это также регистрируется как предупреждение, и пользователю предлагается
// выполнить команду вручную для обновления PATH в текущей сессии.
//
// @param binDir string - Директория, которую нужно добавить в PATH.
// @param log *logger.Logger - Логгер для записи сообщений о ходе выполнения.
func (c *Commands) addPathToEnvironment(binDir string) {
	if err := env.AddToPath(binDir, c.log); err != nil {
		// Не считаем это фатальной ошибкой, продолжаем установку
		mess := fmt.Sprintf("Предупреждение: не удалось добавить директорию в PATH: %v\n", err)
		c.log.Info(mess)
		mess_2 := "Добавьте вручную: " + binDir
		c.log.Info(mess_2)
	} else {
		// Пытаемся обновить PATH в текущей оболочке
		if err := env.UpdateShell(c.log); err != nil {
			mess_1 := fmt.Sprintf("Предупреждение: не удалось обновить PATH в текущей сессии: %v\n", err)
			c.log.Info(mess_1)
			mess_2 := "Выполните вручную: source ~/.zshrc (или source ~/.bash_profile)"
			c.log.Info(mess_2)
		}
	}
}

// createPlistFile создает файл конфигурации для launchd в формате plist.
//
// Функция генерирует XML-файл, который содержит настройки для запуска агента,
// включая путь к исполняемому файлу, параметры запуска и пути к логам.
//
// @param binPath string Абсолютный путь к исполняемому файлу агента
// @return error Ошибка, если не удалось создать или записать файл конфигурации
//
// Пример использования:
//
//	if err := createPlistFile("/usr/local/bin/macbat"); err != nil {
//	    log.Fatalf("Ошибка создания plist: %v", err)
//	}
//
// Примечания:
// - Автоматически создает необходимые директории
// - Устанавливает права доступа 0644 на созданный файл
// - Использует настройки из загруженной конфигурации
func (c *Commands) createPlistFile(binPath string) error {

	// Создаем plist-файл для агента
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>--background</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
	<key>EnvironmentVariables</key>
	<dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>`, paths.AgentIdentifier(), binPath, paths.LogPath(), paths.ErrorLogPath())

	plistPath := paths.PlistPath()
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		mess := fmt.Sprintf("не удалось создать директорию для plist: %v", err)
		c.log.Error(mess)
		return fmt.Errorf("%s", mess)
	}
	if err := utils.CheckWriteAccess(filepath.Dir(plistPath), c.log); err != nil {
		mess := fmt.Sprintf("нет прав на запись в %s: %v", filepath.Dir(plistPath), err)
		c.log.Error(mess)
		return fmt.Errorf("%s", mess)
	}
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		mess := fmt.Sprintf("не удалось записать plist: %v", err)
		c.log.Error(mess)
		return fmt.Errorf("%s", mess)
	} else {
		c.log.Debug(fmt.Sprintf("Plist успешно записан: %s", plistPath))
	}
	return nil
}

// Uninstall выполняет полное удаление приложения из системы.
//
// Процесс удаления включает:
// 1. Остановку и выгрузку агента из launchd
// 2. Удаление plist-файла конфигурации
// 3. Удаление исполняемого файла
// 4. Удаление логов и временных файлов
// 5. Обновление переменной окружения PATH
//
// @return error Ошибка, если процесс удаления не был завершен успешно
//
// Пример использования:
//
//	if err := service.Uninstall(); err != nil {
//	    log.Fatalf("Ошибка удаления: %v", err)
//	}
//
// Примечания:
// - Требует прав администратора
// - Не удаляет пользовательские конфигурации
// - Автоматически обновляет PATH в текущей сессии
func (c *Commands) Uninstall() error {
	c.log.Info("Начало удаления приложения")

	// Создаем менеджер фоновых процессов для их завершения
	bgManager := background.New(c.log)

	// Завершаем все запущенные процессы
	c.log.Info("Завершение фонового процесса...")
	bgManager.Kill("--background")
	c.log.Info("Завершение GUI-агента...")
	bgManager.Kill("--gui-agent")

	// Получаем путь к директории с бинарником перед удалением
	binDir := paths.BinaryPath()

	// Выгружаем агент
	c.log.Info("Отключение агента...")

	// Отключаем и выгружаем агента
	if err := monitor.UnloadAndDisableAgent(c.log); err != nil {
		c.log.Error(fmt.Sprintf("Ошибка отключения агента: %v", err))
	}

	// Удаляем директорию из PATH
	c.removePathFromEnvironment(binDir)

	// Удаляем все оставшиеся файлы
	c.removeAllFiles()

	c.log.Info("Удаление приложения завершено")
	return nil
}

// removePathFromEnvironment удаляет указанную директорию из переменной PATH.
//
// Функция сначала пытается удалить директорию из системной переменной PATH,
// используя внутренний метод RemoveFromPath. Если удаление не удается,
// это регистрируется как предупреждение и выполнение продолжается.
// После успешного удаления функция пытается обновить текущую сессию оболочки
// с помощью внутреннего метода UpdateShell. Если обновление не удается,
// это также регистрируется как предупреждение, и пользователю предлагается
// выполнить команду вручную для обновления PATH в текущей сессии.
//
// @param binDir string - Директория, которую нужно удалить из PATH.
// @param log *logger.Logger - Логгер для записи сообщений о ходе выполнения.
func (c *Commands) removePathFromEnvironment(binDir string) {
	if err := env.RemoveFromPath(binDir, c.log); err != nil {
		// Не считаем это фатальной ошибкой, продолжаем удаление
		mess := fmt.Sprintf("Предупреждение: не удалось удалить директорию из PATH: %v\n", err)
		c.log.Info(mess)
	} else {
		// Пытаемся обновить PATH в текущей оболочке
		if err := env.UpdateShell(c.log); err != nil {
			mess_1 := fmt.Sprintf("Предупреждение: не удалось обновить PATH в текущей сессии: %v\n", err)
			c.log.Info(mess_1)
			mess_2 := "Выполните вручную: source ~/.zshrc (или source ~/.bash_profile)"
			c.log.Info(mess_2)
		}
	}
}

// removeAllFiles удаляет все файлы, используемые приложением.
//
// Функция удаляет файлы, созданные приложением, включая бинарник,
// файл конфигурации, лог-файлы, файл plist и PID-файлы.
//
// @param log *logger.Logger - логгер
func (c *Commands) removeAllFiles() {
	paths := []string{
		paths.BinaryPath(),
		paths.ConfigPath(),
		paths.LogPath(),
		paths.ErrorLogPath(),
		paths.PlistPath(),
	}

	for _, path := range paths {
		c.log.Info(fmt.Sprintf("Удаление файла: %s", path))
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			c.log.Error(fmt.Sprintf("Не удалось удалить %s: %v", path, err))
			// Продолжаем удаление других файлов
		} else if err == nil {
			c.log.Info(fmt.Sprintf("Файл успешно удален: %s", path))
		}
	}
}
