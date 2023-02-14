package main

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/sirupsen/logrus"

	"github.com/integrity-sum/internal/configs"
)

const defaultDBMigrationPath = "file://db/migrations"

func DBMigration(log *logrus.Logger) {
	m, err := migrate.New(defaultDBMigrationPath, configs.GetDBConnString())
	if err != nil {
		log.WithError(err).Fatal("can not create migration")
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			log.WithError(err).Fatal("can not apply migration scripts")
		}
	}
}
