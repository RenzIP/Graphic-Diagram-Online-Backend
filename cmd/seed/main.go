package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/RenzIP/Graphic-Diagram-Online/config"
	"github.com/RenzIP/Graphic-Diagram-Online/db"
	"github.com/RenzIP/Graphic-Diagram-Online/model"
)

func hashPassword(password string) *string {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), 14)
	hash := string(bytes)
	return &hash
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Disconnect(database)

	log.Println("Migrating database tables...")
	err = database.AutoMigrate(
		&model.UserProfile{},
		&model.Workspace{},
		&model.WorkspaceMember{},
		&model.Project{},
		&model.Document{},
		&model.DocumentVersion{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate tables: %v", err)
	}

	log.Println("Seeding database...")

	// Create 10 users
	var users []model.UserProfile
	for i := 1; i <= 10; i++ {
		email := fmt.Sprintf("dummy%d@example.com", i)
		fullName := fmt.Sprintf("Dummy User %d", i)
		role := "user"
		if i == 1 {
			role = "admin"
		}
		username := fmt.Sprintf("dummy%d", i)
		user := model.UserProfile{
			ID:        uuid.New(),
			Username:  &username,
			Password:  hashPassword("password123"),
			Role:      role,
			Email:     &email,
			FullName:  &fullName,
		}
		users = append(users, user)
	}
	
	if err := database.Create(&users).Error; err != nil {
		log.Fatalf("Failed to seed users: %v", err)
	}
	log.Println("Seeded 10 users.")

	// Create 10 workspaces
	var workspaces []model.Workspace
	for i := 1; i <= 10; i++ {
		owner := users[i-1]
		desc := fmt.Sprintf("Workspace description %d", i)
		ws := model.Workspace{
			ID:          uuid.New(),
			Name:        fmt.Sprintf("Dummy Workspace %d", i),
			Slug:        fmt.Sprintf("dummy-workspace-%d-%s", i, uuid.New().String()[:6]),
			OwnerID:     owner.ID,
			Description: &desc,
		}
		workspaces = append(workspaces, ws)
	}

	if err := database.Create(&workspaces).Error; err != nil {
		log.Fatalf("Failed to seed workspaces: %v", err)
	}
	log.Println("Seeded 10 workspaces.")

	// Add users as workspace members
	var members []model.WorkspaceMember
	for i, ws := range workspaces {
		members = append(members, model.WorkspaceMember{
			WorkspaceID: ws.ID,
			UserID:      users[i].ID,
			Role:        "owner",
		})
	}
	if err := database.Create(&members).Error; err != nil {
		log.Fatalf("Failed to seed workspace members: %v", err)
	}

	// Create 10 projects
	var projects []model.Project
	for i := 1; i <= 10; i++ {
		ws := workspaces[i-1]
		ownerID := users[i-1].ID
		desc := fmt.Sprintf("Project description %d", i)
		p := model.Project{
			ID:          uuid.New(),
			WorkspaceID: ws.ID,
			Name:        fmt.Sprintf("Dummy Project %d", i),
			Description: &desc,
			CreatedBy:   &ownerID,
		}
		projects = append(projects, p)
	}

	if err := database.Create(&projects).Error; err != nil {
		log.Fatalf("Failed to seed projects: %v", err)
	}
	log.Println("Seeded 10 projects.")

	// Create 10 documents
	var documents []model.Document
	for i := 1; i <= 10; i++ {
		proj := projects[i-1]
		ws := workspaces[i-1]
		ownerID := users[i-1].ID
		
		doc := model.Document{
			ID:          uuid.New(),
			ProjectID:   &proj.ID,
			WorkspaceID: ws.ID,
			Title:       fmt.Sprintf("Dummy Diagram %d", i),
			DiagramType: "flowchart",
			Content:     model.JSONB(`{"nodes":[],"edges":[]}`),
			View:        model.JSONB(`{"positions":{},"styles":{},"routing":{}}`),
			Version:     1,
			CreatedBy:   &ownerID,
		}
		documents = append(documents, doc)
	}

	if err := database.Create(&documents).Error; err != nil {
		log.Fatalf("Failed to seed documents: %v", err)
	}
	log.Println("Seeded 10 documents.")
	log.Println("Seeding completed successfully!")
}
