package notes

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrNotFound  = errors.New("note not found")
	ErrDuplicate = errors.New("note with this title already exists")
)

type Repo struct {
	col *mongo.Collection
}

// NewRepo создаёт репозиторий и индексы коллекции notes:
//   - уникальный по title;
//   - текстовый по title и content (для $text-поиска);
//   - TTL по expiresAt: документы удаляются, когда наступает expiresAt.
func NewRepo(db *mongo.Database) (*Repo, error) {
	col := db.Collection("notes")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "title", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("title_unique"),
		},
		{
			Keys:    bson.D{{Key: "title", Value: "text"}, {Key: "content", Value: "text"}},
			Options: options.Index().SetName("title_content_text").SetDefaultLanguage("none"),
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("expiresAt_ttl"),
		},
	})
	if err != nil {
		return nil, err
	}
	return &Repo{col: col}, nil
}

// Create добавляет заметку; ttl > 0 включает автоудаление через ttl.
func (r *Repo) Create(ctx context.Context, title, content string, ttl time.Duration) (Note, error) {
	now := time.Now().UTC().Truncate(time.Millisecond) // Mongo хранит время с точностью до миллисекунд
	n := Note{Title: title, Content: content, CreatedAt: now, UpdatedAt: now}
	if ttl > 0 {
		exp := now.Add(ttl)
		n.ExpiresAt = &exp
	}
	res, err := r.col.InsertOne(ctx, n)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return Note{}, ErrDuplicate
		}
		return Note{}, err
	}
	n.ID = res.InsertedID.(primitive.ObjectID)
	return n, nil
}

func (r *Repo) ByID(ctx context.Context, idHex string) (Note, error) {
	oid, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return Note{}, ErrNotFound // некорректный id — для клиента то же, что «нет такой заметки»
	}
	var n Note
	if err := r.col.FindOne(ctx, bson.M{"_id": oid}).Decode(&n); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Note{}, ErrNotFound
		}
		return Note{}, err
	}
	return n, nil
}

// List возвращает заметки от новых к старым (сортировка по _id: ObjectID растёт со временем).
func (r *Repo) List(ctx context.Context, p ListParams) ([]Note, error) {
	filter := bson.M{}
	if p.Q != "" {
		// QuoteMeta: пользовательская строка ищется как текст, а не как регулярное выражение
		filter["title"] = bson.M{"$regex": regexp.QuoteMeta(p.Q), "$options": "i"}
	}
	if p.Search != "" {
		filter["$text"] = bson.M{"$search": p.Search}
	}
	if !p.After.IsZero() {
		filter["_id"] = bson.M{"$lt": p.After}
	}

	opts := options.Find().SetLimit(p.Limit).SetSkip(p.Skip).SetSort(bson.D{{Key: "_id", Value: -1}})
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	out := []Note{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Update частично обновляет заметку: поля с nil не меняются.
func (r *Repo) Update(ctx context.Context, idHex string, title, content *string) (Note, error) {
	oid, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return Note{}, ErrNotFound
	}

	set := bson.M{"updatedAt": time.Now().UTC().Truncate(time.Millisecond)}
	if title != nil {
		set["title"] = *title
	}
	if content != nil {
		set["content"] = *content
	}

	after := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated Note
	err = r.col.FindOneAndUpdate(ctx, bson.M{"_id": oid}, bson.M{"$set": set}, after).Decode(&updated)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Note{}, ErrNotFound
		}
		if mongo.IsDuplicateKeyError(err) {
			return Note{}, ErrDuplicate
		}
		return Note{}, err
	}
	return updated, nil
}

func (r *Repo) Delete(ctx context.Context, idHex string) error {
	oid, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return ErrNotFound
	}
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// Stats считает количество заметок и среднюю/максимальную длину content (aggregation pipeline).
func (r *Repo) Stats(ctx context.Context) (Stats, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "avgContentLength", Value: bson.D{{Key: "$avg", Value: bson.D{{Key: "$strLenCP", Value: "$content"}}}}},
			{Key: "maxContentLength", Value: bson.D{{Key: "$max", Value: bson.D{{Key: "$strLenCP", Value: "$content"}}}}},
		}}},
	}
	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return Stats{}, err
	}
	defer cur.Close(ctx)

	var out []Stats
	if err := cur.All(ctx, &out); err != nil {
		return Stats{}, err
	}
	if len(out) == 0 { // пустая коллекция: $group не вернул ни одной группы
		return Stats{}, nil
	}
	return out[0], nil
}
