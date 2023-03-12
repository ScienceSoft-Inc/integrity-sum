package data

import (
	"context"
	"database/sql"
	"sync"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/configs"
)

type Storage struct {
	db  *sql.DB
	log *logrus.Logger
}

var db *Storage
var openOnce sync.Once

func Open(log *logrus.Logger) (*Storage, error) {
	var err error
	openOnce.Do(func() {
		var conn *sql.DB
		conn, err = ConnectionToDB(log)
		if err != nil {
			return
		}
		db = &Storage{
			db:  conn,
			log: log,
		}
	})
	return db, err
}

func Close() {
	db.db.Close()
}

func DB() *Storage {
	return db
}
func (db *Storage) SQL() *sql.DB {
	return db.db
}

func WithTx(f func(txn *sql.Tx) error) error {
	txn, err := DB().db.Begin()
	if err != nil {
		return err
	}

	if err = f(txn); err != nil {
		if errTx := txn.Rollback(); errTx != nil {
			return errTx
		}
		return err
	}

	if err = txn.Commit(); err != nil {
		return err
	}
	return nil
}

func ExecQueryTx(ctx context.Context, sqlQueryR, sqlQueryH string, argsR []any, argsH ...any) error {
	return WithTx(func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, sqlQueryR, argsR...)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, sqlQueryH, argsH...)
		if err != nil {
			return err
		}
		return nil
	})
}

func ConnectionToDB(logger *logrus.Logger) (*sql.DB, error) {
	logger.Info("Connecting to the database..")
	db, err := sql.Open("postgres", configs.GetDBConnString())
	if err != nil {
		logger.Error("Cannot connect to the database ")
		return nil, err
	}
	return db, nil
}

//// Начало транзакции
//tx, err := db.Begin()
//if err != nil {
//    return err
//}
//
//// Подготовка запроса для вставки данных в таблицу releases
//stmt, err := tx.PrepareQuery("INSERT INTO releases (name, created_at, updated_at, release_type, image) VALUES (?, ?, ?, ?, ?)")
//if err != nil {
//    return err
//}
//
//// Выполнение запроса
//res, err := stmt.Exec(deploymentData.NameDeployment, time.Now(), time.Now(), "", deploymentData.Image)
//if err != nil {
//    return err
//}
//
//// Получение ID вставленной записи
//id, err := res.LastInsertId()
//if err != nil {
//    return err
//}
//
//// Подготовка запроса для вставки данных в таблицу filehashes
//stmt2, err := tx.PrepareQuery("INSERT INTO filehashes (full_file_name, hash_sum, algorithm, name_pod, release_id) VALUES (?, ?, ?, ?, ?)")
//if err != nil {
//    return err
//}
//
//// Выполнение запроса для вставки данных в таблицу filehashes
//_, err = stmt2.Exec(allHashData.FullFileName, allHashData.Hash, allHashData.Algorithm, deploymentData.NamePod, id)
//if err != nil {
//    return err
//}
//
//// Завершение транзакции
//err = tx.Commit()
//if err != nil {
//    return err
//}
//
//return nil
