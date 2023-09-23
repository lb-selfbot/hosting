package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

type User struct {
	ID            int
	Username      string
	IsHosting     bool
	HostingCredit int // In minutes
}

type Database struct {
	Host     string
	Password string

	Conn *sql.DB
}

func NewDatabase() *Database {
	return &Database{
		Host:     os.Getenv("HOST"),
		Password: os.Getenv("PASSWORD"),
	}
}

func (db *Database) Connect() (err error) {
	dsn := fmt.Sprintf("api:%s@tcp(%s:3306)/lightningbot", db.Password, db.Host)

	db.Conn, err = sql.Open("mysql", dsn)
	if err != nil {
		return
	}

	db.Conn.SetConnMaxLifetime(time.Minute * 3)
	db.Conn.SetMaxOpenConns(10)
	db.Conn.SetMaxIdleConns(10)

	return
}

func (db *Database) GetUsers() (users []*User, err error) {
	rows, err := db.Conn.Query("SELECT id, username, is_hosting, hosting_credit FROM api_user")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var u User

		if err := rows.Scan(&u.ID, &u.Username, &u.IsHosting, &u.HostingCredit); err != nil {
			panic(err)
		}

		users = append(users, &u)
		Users[u.ID] = &u
	}

	return
}

func (db *Database) GetUserToken(user *User) (string, error) {
	rows, err := db.Conn.Query("SELECT hosting_token FROM api_user WHERE id = ?", user.ID)
	if err != nil {
		return "", err
	}

	var token string

	for rows.Next() {
		if err := rows.Scan(&token); err != nil {
			panic(err)
		}
	}

	_, err = db.Conn.Exec("UPDATE api_user SET hosting_token = ? WHERE id = ?", "", user.ID)

	return token, err
}

func (db *Database) SaveUser(user *User) error {
	_, err := db.Conn.Exec("UPDATE api_user SET is_hosting = ?, hosting_credit = ? WHERE id = ?", user.IsHosting, user.HostingCredit, user.ID)
	if err != nil {
		return err
	}

	return nil
}
