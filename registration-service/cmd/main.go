package main

import (
	_ "github.com/jackc/pgx/v5/stdlib"
	"registration-service/internal/app"
)

func main() {
	app.Run()
}
