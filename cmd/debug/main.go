package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/RenzIP/Graphic-Diagram-Online/config"
	"github.com/RenzIP/Graphic-Diagram-Online/db"
	"github.com/RenzIP/Graphic-Diagram-Online/dto"
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

	var u model.UserProfile
	if err := database.First(&u, "email = ?", "nurfadilahmfauzi@gmail.com").Error; err != nil {
		fmt.Printf("User not found: %v\n", err)
		return
	}

	// Just print the response object!
	resp := &dto.AuthMeResp{
		ID:        u.ID.String(),
		Name:      u.Name,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Provider:  u.Provider,
		Avatar:    u.Avatar,
		Status:    u.Status,
		FullName:  u.FullName,
		AvatarURL: u.AvatarURL,
	}

	b, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println("API Response JSON:")
	fmt.Println(string(b))

}
