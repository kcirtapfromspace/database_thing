package main

import (
	"database/sql"
	"database_thing/pkg/config"
	"database_thing/pkg/db"
	"database_thing/pkg/filepkg"
	"database_thing/pkg/web"
	"fmt"
	"log"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		config.Module,
		db.Module,
		filepkg.Module,
		web.Module,
		fx.Invoke(func(cd config.ConfigData, d *sql.DB) error {
			if cd.Err != nil {
				log.Printf("Error loading config: %v", cd.Err)
				return cd.Err
			}
			server := web.ListenAndServe(cd.Config.ListenAddr)
			if server == nil {
				log.Fatal("Error starting server")
				return fmt.Errorf("Error starting server")
			}
			return nil
		}),
	)
	app.Run()
}
