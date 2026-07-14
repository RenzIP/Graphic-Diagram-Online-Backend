package main

import (
	"fmt"
	"log"

	"github.com/RenzIP/Graphic-Diagram-Online/config"
	"github.com/RenzIP/Graphic-Diagram-Online/db"
	"github.com/RenzIP/Graphic-Diagram-Online/model"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer db.Disconnect(database)

	var users []model.UserProfile
	if err := database.Find(&users).Error; err != nil {
		fmt.Printf("Failed to find users: %v\n", err)
	} else {
		for _, u := range users {
			email := "<nil>"
			if u.Email != nil {
				email = *u.Email
			}
			username := "<nil>"
			if u.Username != nil {
				username = *u.Username
			}
            name := "<nil>"
            if u.Name != nil {
                name = *u.Name
            }
			fmt.Printf("User: ID=%s Provider=%s Email=%s Username=%s Name=%s\n", u.ID, u.Provider, email, username, name)
		}
	}
}
