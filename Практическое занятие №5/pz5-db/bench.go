package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// runBench измеряет, как настройки пула влияют на время выполнения запросов.
func runBench(dsn string) {
	const (
		total       = 200 // запросов в каждом замере
		concurrency = 50  // параллельных горутин
	)

	fmt.Printf("== 1. MaxOpenConns: %d запросов, %d горутин ==\n", total, concurrency)
	fmt.Printf("%-12s %-22s %s\n", "MaxOpen", "SELECT pg_sleep(5мс)", "SELECT 1")
	for _, n := range []int{1, 2, 5, 10, 20, 50} {
		slow := measureParallel(dsn, n, n, total, concurrency, "SELECT pg_sleep(0.005)")
		fast := measureParallel(dsn, n, n, total, concurrency, "SELECT 1")
		fmt.Printf("%-12d %-22s %s\n", n, slow.Round(time.Millisecond), fast.Round(time.Millisecond))
	}

	fmt.Printf("\n== 2. MaxIdleConns: 300 последовательных SELECT 1 ==\n")
	fmt.Printf("%-12s %s\n", "MaxIdle", "время")
	for _, idle := range []int{0, 1, 5} {
		d := measureParallel(dsn, 10, idle, 300, 1, "SELECT 1")
		fmt.Printf("%-12d %s\n", idle, d.Round(time.Millisecond))
	}
}

// measureParallel открывает свежий пул с заданными лимитами и выполняет
// total запросов в concurrency горутинах; возвращает общее время.
func measureParallel(dsn string, maxOpen, maxIdle, total, concurrency int, query string) time.Duration {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)

	ctx := context.Background()
	// прогрев: одно соединение, чтобы не мерить первый коннект
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	jobs := make(chan struct{}, total)
	for i := 0; i < total; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				rows, err := db.QueryContext(ctx, query)
				if err != nil {
					log.Fatal(err)
				}
				rows.Close()
			}
		}()
	}
	wg.Wait()
	return time.Since(start)
}
