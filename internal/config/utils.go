package config

// import (
// 	"fmt"
// 	"macbat/internal/log"
// 	"macbat/internal/utils"
// 	"os"
// )

// func LoadFileConfig(box *utils.WindowBuffer) (*Config, error) {
// 	// Загрузка конфигурации из файла config.json
// 	log.Info("Загрузка конфигурации...")
// 	cfg, err := LoadConfig()
// 	if err != nil {
// 		errMsg := fmt.Sprintf("Ошибка загрузки конфигурации: %v", err)
// 		log.Error(errMsg)
// 		box.AddLine(errMsg, "", "")
// 		box.PrintBox()
// 		os.Exit(1)
// 	}
// 	log.Info("Конфигурация успешно загружена")
// 	return cfg, nil
// }
