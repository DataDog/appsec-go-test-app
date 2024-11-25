package vulnerable

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"

	_ "modernc.org/sqlite"
)

const tables = `
CREATE TABLE user (
   id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
   name  text NOT NULL,
   email text NOT NULL,
   password text NOT NULL
);
CREATE TABLE product (
   id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
   name  text NOT NULL,
   category  text NOT NULL,
   price  int NOT NULL
);
`

func PrepareSQLDB(nbEntries int) (*sql.DB, error) {
	fp, err := os.CreateTemp("", "go-dvwa-*")
	if err != nil {
		return nil, err
	}

	defer fp.Close()

	prepareDB, err := sql.Open("sqlite", fp.Name())
	if err != nil {
		log.Fatalln("unexpected sql.Open error:", err)
	}

	defer prepareDB.Close()

	if _, err := prepareDB.Exec(tables); err != nil {
		return nil, err
	}

	for i := 0; i < nbEntries; i++ {
		_, err := prepareDB.Exec(
			"INSERT INTO user (name, email, password) VALUES (?, ?, ?)",
			fmt.Sprintf("User#%d", i),
			fmt.Sprintf("user%d@mail.com", i),
			fmt.Sprintf("secret-password#%d", i))
		if err != nil {
			return nil, err
		}

		_, err = prepareDB.Exec(
			"INSERT INTO product (name, category, price) VALUES (?, ?, ?)",
			fmt.Sprintf("Product %d", i),
			"sneaker",
			rand.Intn(500))
		if err != nil {
			return nil, err
		}
	}

	// Reopen the database to enable tracing
	db, err := sql.Open("sqlite", fp.Name())
	if err != nil {
		log.Fatalln("unexpected rerunning sql.Open error:", err)
	}

	return db, nil
}

type (
	Product struct {
		Id       int
		Name     string
		Category string
		Price    string
	}
	User struct {
		Name, Password, Email string
	}
)

func GetProducts(_ context.Context, db *sql.DB, category string) ([]Product, error) {
	rows, err := db.Query("SELECT * FROM product WHERE category='" + category + "'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []Product
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.Id, &product.Name, &product.Category, &product.Price); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

func GetUser(ctx context.Context, db *sql.DB, username string) (*User, error) {
	rows, err := db.QueryContext(ctx, "SELECT name, email, password FROM user WHERE name='"+username+"'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var user User
		if err := rows.Scan(&user.Name, &user.Email, &user.Password); err != nil {
			return nil, err
		}
		return &user, nil
	}
	return nil, errors.New("Could not find user " + username)
}

func AddUser(ctx context.Context, db *sql.DB, username string, password string) {
	db.ExecContext(ctx, "INSERT INTO user (name, email, password) VALUES (?, ?, ?)", username, username+"@"+"bogus.com", password)
}
