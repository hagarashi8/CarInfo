package main

import (
	"effMob/cars"
	"effMob/api"
	"effMob/default_router"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	logrus "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)



type EnvVars struct {
	CarInfoUrl string
	PGUrl string
	PGUsername string
	PGPassword string
	PGDatabase string
	PGSSLMode bool
	LogLevel logrus.Level
}

func ReadEnvVars() (vars *EnvVars, err error) {
	vars = new(EnvVars)

	level := os.Getenv("LOG_LEVEL")
	vars.LogLevel, err = logrus.ParseLevel(level)
	if err != nil {
		vars.LogLevel = logrus.InfoLevel
	}
	
	vars.CarInfoUrl = os.Getenv("CAR_SERVICE_URL")
	if vars.CarInfoUrl == "" {
		err = fmt.Errorf("Не указан URL сервиса информации о машинах")
		vars = nil
		return
	}

	vars.PGUrl = os.Getenv("PG_URL")
	if vars.PGUrl == "" {
		err = fmt.Errorf("Не указан URL Postgres")
		vars = nil
		return
	}

	vars.PGDatabase = os.Getenv("PG_DATABASE")
	if vars.PGDatabase == "" {
		err = fmt.Errorf("Не указана база данных Postgres")
		vars = nil
		return
	}

	vars.PGUsername = os.Getenv("PG_USERNAME")
	if vars.PGUrl == "" {
		err = fmt.Errorf("Не указано имя пользоватля Postgres")
		vars = nil
		return
	}

	vars.PGPassword = os.Getenv("PG_PASSWORD")
	

	if os.Getenv("PG_SSL_MODE") == "true" {
		vars.PGSSLMode = true
	} else {
		vars.PGSSLMode = false
	}
	return
}

func (self *EnvVars) GetConnectionString() string {
	ssl := "disabled"
	if self.PGSSLMode {
		ssl = "enabled"
	}
	return fmt.Sprintf(
		"host=%s user=%s dbname=%s password=%s sslmode=%s", 
		self.PGUrl, 
		self.PGUsername, 
		self.PGDatabase, 
		self.PGPassword,
		ssl,
	)
}

func SetupAPIClient(env *EnvVars) (client *api.APIClient){
	config := api.NewConfiguration()
	config.BasePath = env.CarInfoUrl
	client = api.NewAPIClient(config)
	return
}

func InitDBConnection(env *EnvVars) (db *gorm.DB, err error){
	dsn := env.GetConnectionString()
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { 
		return
	}

	err = db.AutoMigrate(&cars.Car{})
	if err != nil {
		db = nil
		return
	}
	return
}

// @title Car Info++
// @version 0.0.1
func main() {
	// Настройка логгинга
	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)
	log.SetOutput(os.Stderr)
	// Загрузка переменных окружения
	err := godotenv.Load()
	if err != nil {
		log.WithError(err).Warnln(".env файл не загружен. Используются только переменные окружения.")
	}
	env, err := ReadEnvVars()
	if err != nil {
		log.WithError(err).Fatalln("Переменные среды не загружены")
	}
	log.Debugln(*env)
	log.Traceln("Переменные окружения загружены")
	// Создание API клиента
	client := SetupAPIClient(env)
	log.Traceln("API Клиент создан")
	// Создание соединения с базой данных
	db, err := InitDBConnection(env)
	if err != nil {
		log.WithError(err).Fatalln("Не получилось создать соединения с базой данных")
	}
	log.Traceln("Соединение с базой данных создано")
	err = http.ListenAndServe(":8080", defaultrouter.SetupMux(client, db, log))
	if err != nil {
		log.WithError(err).Fatalln("Сервер завершил работу с ошибкой")
	}
}

