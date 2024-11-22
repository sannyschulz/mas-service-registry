package main

// DB connection
// use sqlite3 as the database
// import the sqlite3 driver
import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// path to the database file
const dbPath = "./data/strudyref.sqlite3"

func main() {

	var db *sql.DB
	// check if the database path exists
	_, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		db, err = createDB(dbPath)
		if err != nil {
			log.Panic(err)
		}
	} else {
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Panic(err)
		}
		defer db.Close()
	}

	// check if the database is alive
	if err := db.Ping(); err != nil {
		log.Panic(err)
	}

}

func createDB(dbPath string) (*sql.DB, error) {
	// create the database file
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// create the tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sturdyrefs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sturdyRef TEXT NOT NULL,
			serviceId TEXT NOT NULL,
			payload TEXT NOT NULL,
			authToken TEXT NOT NULL
		);
	`)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// add sturdyref to the database
func addSturdyRef(db *sql.DB, sturdyRef, serviceId, payload, authToken string) error {
	sqlStmt := `INSERT INTO sturdyrefs (sturdyRef, serviceId, payload, authToken)
		VALUES (?, ?, ?, ?)`

	transaction, err := db.Begin()
	if err != nil {
		return err
	}
	pStmt, err := transaction.Prepare(sqlStmt)
	if err != nil {
		return err
	}
	defer pStmt.Close()
	pStmt.Exec(sturdyRef, serviceId, payload, authToken)
	err = transaction.Commit()
	if err != nil {
		return err
	}
	return nil
}

// prepared statement to get sturdyref from the database
func getSturdyRef(db *sql.DB, sturdyRef string) (string, string, string, error) {
	var serviceId, payload, authToken string
	err := db.QueryRow(`
		SELECT serviceId, payload, authToken
		FROM sturdyrefs
		WHERE sturdyRef = ?;
	`, sturdyRef).Scan(&serviceId, &payload, &authToken)
	return serviceId, payload, authToken, err
}
