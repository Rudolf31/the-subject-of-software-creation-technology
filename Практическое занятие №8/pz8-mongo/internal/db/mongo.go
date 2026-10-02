package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDeps struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// ConnectMongo подключается к MongoDB и проверяет соединение командой ping.
// Если сервер недоступен, ошибка вернётся примерно через serverSelection секунд.
func ConnectMongo(ctx context.Context, uri, dbName string) (*MongoDeps, error) {
	const serverSelection = 5 * time.Second

	opts := options.Client().ApplyURI(uri).SetServerSelectionTimeout(serverSelection)

	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// mongo.NewClient + Connect считаются устаревшими: mongo.Connect делает то же самое одной функцией
	cli, err := mongo.Connect(dialCtx, opts)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	pingCtx, cancelPing := context.WithTimeout(ctx, 3*time.Second+serverSelection)
	defer cancelPing()
	if err := cli.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = cli.Disconnect(ctx)
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &MongoDeps{Client: cli, Database: cli.Database(dbName)}, nil
}
