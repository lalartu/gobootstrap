package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Database DatabaseConfig
	Settings SettingsConfig
	Logging  LoggingConfig
}

type DatabaseConfig struct {
	Server   string
	Port     int
	Username string
	Password string
}

type SettingsConfig struct {
	File      string
	Delimiter string
	Length    int
}

type LoggingConfig struct {
	Db    string
	Table string
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"

func GenerateRandomPassword(length int) string {
	rand.Seed(time.Now().UnixNano())
	password := make([]byte, length)
	for i := range password {
		password[i] = charset[rand.Intn(len(charset))]
	}
	return string(password)
}

func createUsers(filename string, db *sql.DB, config Config) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("Failed to open file")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		parts := strings.Split(line, " ")
		password := GenerateRandomPassword(config.Settings.Length)
		name := parts[0] + config.Settings.Delimiter + parts[1]

		db.Query(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", name))
		db.Query(fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", name, password))
		db.Query(fmt.Sprintf("GRANT USAGE ON *.* TO '%s'@'%%'", name))
		db.Query(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%' WITH GRANT OPTION", name, name))
		db.Query("FLUSH PRIVILEGES")
		db.Query(fmt.Sprintf("INSERT INTO `%s`.`%s` (`id`, `username`, `password`) VALUES (NULL, '%s', '%s')", config.Logging.Db, config.Logging.Table, name, password))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("Error reading file")
	}
}

func createUsersNoConfig(filename string, db *sql.DB, pwLen int, delimiter string, loggingDb string, loggingTable string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("Failed to open file")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		parts := strings.Split(line, " ")
		password := GenerateRandomPassword(pwLen)
		name := parts[0] + delimiter + parts[1]

		db.Query(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", name))
		db.Query(fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", name, password))
		db.Query(fmt.Sprintf("GRANT USAGE ON *.* TO '%s'@'%%'", name))
		db.Query(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%' WITH GRANT OPTION", name, name))
		db.Query("FLUSH PRIVILEGES")
		db.Query(fmt.Sprintf("INSERT INTO `%s`.`%s` (`id`, `username`, `password`) VALUES (NULL, '%s', '%s')", loggingDb, loggingTable, name, password))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("Error reading file")
	}
}

func main() {
	var (
		useConfig    bool
		config       Config
		databaseIp   string
		databasePort string
		databaseUser string
		databasePass string
		lenStr       string
		delimiter    string
		loggingDb    string
		loggingTable string
		filePath     string
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[bool]().
				Title("Do you want to use Connection parameters in the config?").
				Options(
					huh.NewOption("Yes", true),
					huh.NewOption("No", false),
				).
				Value(&useConfig),
		),
	)

	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}

	if useConfig {
		if _, err := toml.DecodeFile("config.toml", &config); err != nil {
			log.Fatal(err)
		}

		if config.Settings.File != "" {

			db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/", config.Database.Username, config.Database.Password, config.Database.Server, config.Database.Port))

			if err != nil {
				log.Fatal(err)
			}

			defer db.Close()

			if config.Logging.Db != "" && config.Logging.Table != "" {
				db.Query(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", config.Logging.Db))
				db.Query(fmt.Sprintf("CREATE TABLE `%s`.`%s` (`id` INT NOT NULL AUTO_INCREMENT , `username` VARCHAR(200) NOT NULL , `password` VARCHAR(200) NOT NULL , PRIMARY KEY (`id`))", config.Logging.Db, config.Logging.Table))
			}

			_ = spinner.New().Title("Creating users").Accessible(true).Run()
			createUsers(config.Settings.File, db, config)
			fmt.Println("Done!")

		}

	} else {
		formSql := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Insert the database IP or domain").
					Value(&databaseIp),

				huh.NewInput().
					Title("Insert the database Port").
					Value(&databasePort),

				huh.NewInput().
					Title("Insert the database Username").
					Value(&databaseUser),

				huh.NewInput().
					Title("Insert the database Password").
					EchoMode(huh.EchoModePassword).
					Value(&databasePass),

				huh.NewInput().
					Title("Insert the desidered length for user password").
					Value(&lenStr),

				huh.NewInput().
					Title("Insert the delimiter of the user format").
					Placeholder("_").
					Value(&delimiter),

				huh.NewInput().
					Title("Insert the name of the database you want to be used for logging").
					Value(&loggingDb),

				huh.NewInput().
					Title("Insert the name of the table you want to be used for logging").
					Value(&loggingTable),

				huh.NewInput().
					Title("Insert the file path of the file with student names").
					Value(&filePath),
			),
		)

		err := formSql.Run()
		if err != nil {
			log.Fatal(err)
		}

		dbPort, err := strconv.Atoi(databasePort)
		if err != nil {
			log.Fatal(err)
		}

		pwLength, err := strconv.Atoi(lenStr)
		if err != nil {
			log.Fatal(err)
		}

		db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/", databaseUser, databasePass, databaseIp, dbPort))

		if err != nil {
			log.Fatal(err)
		}

		defer db.Close()

		if loggingDb != "" && loggingTable != "" {
			db.Query(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", loggingDb))
			db.Query(fmt.Sprintf("CREATE TABLE `%s`.`%s` (`id` INT NOT NULL AUTO_INCREMENT , `username` VARCHAR(200) NOT NULL , `password` VARCHAR(200) NOT NULL , PRIMARY KEY (`id`))", loggingDb, loggingTable))
		}

		_ = spinner.New().Title("Creating users").Accessible(true).Run()
		createUsersNoConfig(filePath, db, pwLength, delimiter, loggingDb, loggingTable)
		fmt.Println("Done!")

	}

}
