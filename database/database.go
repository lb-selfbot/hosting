package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

type User struct {
	ID              int
	Username        string
	IsHosting       bool
	HostingCredit   int // In minutes
	HostingLifetime bool
}

type Database struct {
	Host     string
	Username string
	Password string

	Conn *sql.DB
}

// ID --> User
var Users = make(map[int]*User)

func NewDatabase() *Database {
	return &Database{
		Host:     os.Getenv("DB_HOST"),
		Username: os.Getenv("DB_USERNAME"),
		Password: os.Getenv("DB_PASSWORD"),
	}
}

func (db *Database) Connect() (err error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/lightningbot", db.Username, db.Password, db.Host)

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
	rows, err := db.Conn.Query("SELECT id, username, is_hosting, hosting_credit, hosting_lifetime FROM api_user")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var u User

		if err := rows.Scan(&u.ID, &u.Username, &u.IsHosting, &u.HostingCredit, &u.HostingLifetime); err != nil {
			return nil, err
		}

		users = append(users, &u)
		Users[u.ID] = &u
	}

	return
}

func (db *Database) GetUserByID(id int) (*User, error) {
	rows, err := db.Conn.Query("SELECT id, username, is_hosting, hosting_credit, hosting_lifetime FROM api_user WHERE id = ?", id)
	if err != nil {
		return nil, err
	}

	var user User

	for rows.Next() {
		if err := rows.Scan(&user.ID, &user.Username, &user.IsHosting, &user.HostingCredit, &user.HostingLifetime); err != nil {
			return nil, err
		}
	}

	return &user, nil
}

func (db *Database) GetUserToken(user *User) (string, error) {
	rows, err := db.Conn.Query("SELECT hosting_token FROM api_user WHERE id = ?", user.ID)
	if err != nil {
		return "", err
	}

	var token string

	for rows.Next() {
		if err := rows.Scan(&token); err != nil {
			return "", err
		}
	}

	return token, nil
}

func (db *Database) SaveUser(user *User) error {
	_, err := db.Conn.Exec("UPDATE api_user SET is_hosting = ?, hosting_credit = ? WHERE id = ?", user.IsHosting, user.HostingCredit, user.ID)
	if err != nil {
		return err
	}

	return nil
}
