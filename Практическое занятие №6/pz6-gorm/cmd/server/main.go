package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Rudolf31/pz6-gorm/internal/db"
	"github.com/Rudolf31/pz6-gorm/internal/httpapi"
	"github.com/Rudolf31/pz6-gorm/internal/models"
)

func main() {
	d, err := db.Connect(os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatal(err)
	}

	// Автоматически создаст (или обновит) таблицы под наши модели
	if err := d.AutoMigrate(&models.User{}, &models.Note{}, &models.Tag{}); err != nil {
		log.Fatal("migrate:", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := httpapi.BuildRouter(d)

	log.Println("listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
