package notes

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Note struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"       json:"id"`
	Title     string             `bson:"title"               json:"title"`
	Content   string             `bson:"content"             json:"content"`
	CreatedAt time.Time          `bson:"createdAt"           json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"           json:"updatedAt"`
	// ExpiresAt — момент автоматического удаления (TTL-индекс); nil — хранить вечно.
	ExpiresAt *time.Time `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
}

// Stats — результат агрегации по коллекции.
type Stats struct {
	Count            int64   `bson:"count"            json:"count"`
	AvgContentLength float64 `bson:"avgContentLength" json:"avgContentLength"`
	MaxContentLength int64   `bson:"maxContentLength" json:"maxContentLength"`
}

// ListParams — параметры выборки списка.
type ListParams struct {
	Q      string             // поиск по заголовку (подстрока, без учёта регистра)
	Search string             // полнотекстовый поиск по title и content ($text)
	Limit  int64              // размер страницы
	Skip   int64              // пагинация «страницами»
	After  primitive.ObjectID // пагинация по курсору: только заметки старше этого id
}
