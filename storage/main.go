package main

// DB connection
// use sqlite3 as the database
// import the sqlite3 driver
import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	commonlib "github.com/zalf-rpm/mas-infrastructure/src/go/commonlib"
)

func main() {

	configPath := flag.String("config", "", "config file")
	configGen := flag.Bool("config-gen", false, "generate a config file")
	flag.Parse()

	// read the config file, if it exists
	var config *commonlib.Config
	var err error
	if *configGen {
		gen := &ConfigConfiguratorImpl{}
		// generate a config file if it does not exist yet
		config, err = commonlib.ConfigGen(*configPath, gen)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Config file generated at:", *configPath)
	} else {
		config, err = commonlib.ReadConfig(*configPath, nil)
		if err != nil {
			log.Fatal(err)
		}
	}

	var db *sql.DB
	dbPath := config.Data["Database"].(map[string]interface{})["Path"].(string)
	// check if the database path exists
	_, err = os.Stat(dbPath)
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

	// create a channel for db requests
	requestChan := make(chan dbRequest)
	go dbRequestScheduler(db, requestChan)

	// create a channel for db responses
	responseChan := make(chan dbResponse)

	// listen to incoming requests
	listenForRequests(requestChan, responseChan, config)

}

// db request sheduler
func dbRequestScheduler(db *sql.DB, requestChan <-chan dbRequest) {
	for request := range requestChan {
		switch request.requestType {
		case addSturdyRefRequest:
			err := addSturdyRef(db, request.sturdyRef, request.serviceId, request.payload, request.authToken)
			request.responseChan <- dbResponse{err: err}
		case getSturdyRefRequest:
			serviceId, payload, authToken, err := getSturdyRef(db, request.sturdyRef)
			request.responseChan <- dbResponse{sturdyRefs: []*sturdyRefStored{{payload: payload, serviceId: serviceId, authToken: authToken}}, err: err}
		case deleteSturdyRefRequest:
			err := deleteSturdyRef(db, request.sturdyRef)
			request.responseChan <- dbResponse{err: err}
		case listSturdyRefsRequest:
			sturdyRefs, err := listSturdyRefs(db)
			request.responseChan <- dbResponse{sturdyRefs: sturdyRefs, err: err}
		case listSturdyRefsByAuthTokenRequest:
			sturdyRefs, err := listSturdyRefsByAuthToken(db, request.authToken)
			request.responseChan <- dbResponse{sturdyRefs: sturdyRefs, err: err}
		}
	}
}

// db request type
type dbRequestType int

const (
	addSturdyRefRequest dbRequestType = iota
	getSturdyRefRequest
	deleteSturdyRefRequest
	listSturdyRefsRequest
	listSturdyRefsByAuthTokenRequest
)

// db request
type dbRequest struct {
	requestType  dbRequestType
	sturdyRef    string
	serviceId    string
	payload      string
	authToken    string
	responseChan chan dbResponse
}
type sturdyRefStored struct {
	sturdyRef string
	serviceId string
	payload   string
	authToken string
}

// db response
type dbResponse struct {
	sturdyRefs []*sturdyRefStored
	err        error
}

func createDB(dbPath string) (*sql.DB, error) {
	// create the database file
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

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

// prepared statement to delete sturdyref from the database
func deleteSturdyRef(db *sql.DB, sturdyRef string) error {
	_, err := db.Exec(`
		DELETE FROM sturdyrefs
		WHERE sturdyRef = ?;
	`, sturdyRef)
	return err
}

// list all sturdyrefs in the database
func listSturdyRefs(db *sql.DB) ([]*sturdyRefStored, error) {

	rows, err := db.Query(`
		SELECT sturdyRef, serviceId, payload, authToken
		FROM sturdyrefs;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sturdyRefs := []*sturdyRefStored{}
	for rows.Next() {
		sturdyRefStored := &sturdyRefStored{}
		err = rows.Scan(
			&sturdyRefStored.sturdyRef,
			&sturdyRefStored.serviceId,
			&sturdyRefStored.payload,
			&sturdyRefStored.authToken)
		if err != nil {
			return nil, err
		}
		sturdyRefs = append(sturdyRefs, sturdyRefStored)
	}
	return sturdyRefs, nil
}

// list sturdyrefs by auth token
func listSturdyRefsByAuthToken(db *sql.DB, authToken string) ([]*sturdyRefStored, error) {
	rows, err := db.Query(`
		SELECT sturdyRef, serviceId, payload
		FROM sturdyrefs
		WHERE authToken = ?;
	`, authToken)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sturdyRefs := []*sturdyRefStored{}
	for rows.Next() {
		sturdyRefStored := &sturdyRefStored{}
		err = rows.Scan(
			&sturdyRefStored.sturdyRef,
			&sturdyRefStored.serviceId,
			&sturdyRefStored.payload)
		if err != nil {
			return nil, err
		}
		sturdyRefs = append(sturdyRefs, sturdyRefStored)
	}
	return sturdyRefs, nil
}

type ConfigConfiguratorImpl struct {
}

func (c *ConfigConfiguratorImpl) GetDefaultConfig() *commonlib.Config {
	defaultConfig := commonlib.DefaultConfig()
	defaultConfig.Data["Service"].(map[string]interface{})["Name"] = "Storage Service"
	defaultConfig.Data["Service"].(map[string]interface{})["Id"] = "storage_service"
	defaultConfig.Data["Service"].(map[string]interface{})["Port"] = 0 // use any free port
	defaultConfig.Data["Service"].(map[string]interface{})["Host"] = "localhost"
	defaultConfig.Data["Service"].(map[string]interface{})["Description"] = "store sturdyref service"
	defaultConfig.Data["Database"].(map[string]interface{})["Type"] = "sqlite3"
	defaultConfig.Data["Database"].(map[string]interface{})["Path"] = "./data/sturdyref.sqlite3"

	return defaultConfig
}
