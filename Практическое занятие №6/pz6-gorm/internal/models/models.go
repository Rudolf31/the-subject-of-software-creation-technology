package models

import "time"

// User — автор заметок (1 пользователь → много заметок).
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Email     string    `gorm:"size:200;uniqueIndex;not null" json:"email"`
	Notes     []Note    `json:"notes,omitempty"` // 1:N
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Note — заметка: принадлежит пользователю, имеет много тегов.
type Note struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:200;not null" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	UserID    uint      `gorm:"not null;index" json:"userId"`
	User      *User     `json:"user,omitempty"`
	Tags      []Tag     `gorm:"many2many:note_tags;" json:"tags"` // M:N
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Tag — тег (многие ко многим с заметками через таблицу note_tags).
type Tag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Notes     []Note    `gorm:"many2many:note_tags;" json:"notes,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
