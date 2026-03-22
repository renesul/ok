package integration

import (
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/renesul/ok/application"
	"github.com/renesul/ok/infrastructure/database"
	"github.com/renesul/ok/infrastructure/repository"
	"github.com/renesul/ok/interfaces/http"
	"github.com/renesul/ok/interfaces/http/handler"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	testDB  *gorm.DB
	testApp *fiber.App
	dbPath  string
)

func TestMain(m *testing.M) {
	dbPath = "test_ok.db"

	var err error
	testDB, err = database.New(dbPath, false)
	if err != nil {
		panic("open test database: " + err.Error())
	}

	if err := database.RunMigrations(testDB); err != nil {
		panic("run test migrations: " + err.Error())
	}

	log := zap.NewNop()

	userRepository := repository.NewUserRepository(testDB, log)
	userService := application.NewUserService(userRepository, log)
	userHandler := handler.NewUserHandler(userService, log)
	testApp = http.NewServer(userHandler, log)

	code := m.Run()

	os.Remove(dbPath)
	os.Exit(code)
}

func cleanupUsers(t *testing.T) {
	t.Helper()
	testDB.Exec("DELETE FROM users")
}
