package main

import (
	"fmt"
	"os"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PeerStatus string

const (
	PeerStatusPending  PeerStatus = "pending"
	PeerStatusCreated  PeerStatus = "created"
	PeerStatusDeleting PeerStatus = "deleting"
)

type AccessRuleStatus string

const (
	AccessRuleStatusPending AccessRuleStatus = "pending"
	AccessRuleStatusCreated AccessRuleStatus = "created"
)

type Peer struct {
	ID         string     `gorm:"type:uuid;primary_key" json:"id"`
	IP         string     `gorm:"type:varchar(255);index" json:"ip"`
	PublicKey  string     `gorm:"type:text" json:"public_key"`
	PrivateKey string     `gorm:"type:text" json:"private_key"`
	Status     PeerStatus `gorm:"type:varchar(20);index" json:"status"`
}

type AccessRule struct {
	ID      string           `gorm:"type:uuid;primary_key" json:"id"`
	PeerAID string           `gorm:"type:uuid;index:idx_peer_id" json:"peer_a_id"`
	PeerBID string           `gorm:"type:uuid;index:idx_peer_id" json:"peer_b_id"`
	Status  AccessRuleStatus `gorm:"type:varchar(20);index" json:"status"`
}

var (
	db   *gorm.DB
	once sync.Once
)

const dbPath = "pikotunnel.db"

// GetDB returns the database instance, creating it if necessary
func GetDB() *gorm.DB {
	once.Do(func() {
		// Create SQLite database file if it doesn't exist
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			file, err := os.Create(dbPath)
			if err != nil {
				panic(fmt.Sprintf("failed to create database file: %v", err))
			}
			file.Close()
		}

		var err error
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			panic("failed to connect database > " + err.Error())
		}

		// Enable WAL mode
		sqlDB, err := db.DB()
		if err != nil {
			panic("failed to get generic database object")
		}

		_, err = sqlDB.Exec("PRAGMA journal_mode = WAL;")
		if err != nil {
			panic(fmt.Sprintf("failed to enable WAL mode: %v", err))
		}

		// Auto migrate the schemas
		err = db.AutoMigrate(&Peer{}, &AccessRule{})
		if err != nil {
			panic("failed to migrate database")
		}
	})
	return db
}

// CloseDB closes the database connection
func CloseDB() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
