package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/qadam/backend/internal/config"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := config.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	userRepo := repositories.NewUserRepository(pool)
	roleRepo := repositories.NewRoleRepository(pool)
	userAdminService := services.NewUserAdminService(userRepo, roleRepo)

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	fmt.Print("Full name: ")
	fullName, _ := reader.ReadString('\n')
	fullName = strings.TrimSpace(fullName)

	if username == "" || password == "" || fullName == "" {
		log.Fatal("username, password and full name are required")
	}

	// CLI-создание первого admin'а: actorID = созданный пользователь
	// (сам себя); audit-сервис в CLI не подключён — запись не создаётся.
	user, err := userAdminService.Create(ctx, "", services.CreateUserInput{
		Username: username,
		Password: password,
		FullName: fullName,
		Role:     models.RoleAdmin,
		Language: "ru",
		Theme:    "dark",
	})
	if err != nil {
		log.Fatalf("failed to create admin: %v", err)
	}

	fmt.Println()
	fmt.Println("Admin created successfully!")
	fmt.Printf("ID:       %s\n", user.ID)
	fmt.Printf("Username: %s\n", user.Username)
	fmt.Printf("Full name: %s\n", user.FullName)
	fmt.Printf("Role:     %s\n", user.RoleKey)
}
