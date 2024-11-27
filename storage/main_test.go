package main

import (
	"database/sql"
	"os"
	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func Test_createDB(t *testing.T) {
	type args struct {
		dbPath string
	}
	tests := []struct {
		name    string
		args    args
		want    *sql.DB
		wantErr bool
	}{
		{"Creation Test", args{"test/test_create.db"}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// check if db already exists
			_, err := os.Stat(tt.args.dbPath)
			if err == nil {
				// remove db if it exists from a previous test
				err = os.Remove(tt.args.dbPath)
				if err != nil {
					t.Error(err)
					t.Errorf("Error deleting db")
				}
			}
			got, err := createDB(tt.args.dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("createDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// query to check if table exists
			_, err = got.Query("SELECT * FROM sturdyrefs")
			if err != nil {
				t.Error(err)
				t.Errorf("Table not created")
			}

			if got != nil {
				err := got.Close()
				if err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func Test_addSturdyRef(t *testing.T) {

	db, err := setupdb(t, "test/test_add.db")
	if err != nil {
		return
	}
	defer db.Close()

	type args struct {
		db        *sql.DB
		sturdyRef string
		serviceId string
		payload   string
		authToken string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Add Test", args{db, "test_sturdy_ref", "test_service_id", "test_payload", "test_auth_token"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := addSturdyRef(tt.args.db, tt.args.sturdyRef, tt.args.serviceId, tt.args.payload, tt.args.authToken); (err != nil) != tt.wantErr {
				t.Errorf("addSturdyRef() error = %v, wantErr %v", err, tt.wantErr)
			}
			// query to check if the data is added
			rows, err := tt.args.db.Query("SELECT * FROM sturdyrefs WHERE sturdyRef = ?", tt.args.sturdyRef)
			if err != nil {
				t.Error(err)
			}
			defer rows.Close()
			if rows.Next() {
				var sturdyRef, serviceId, payload, authToken string
				var id int
				err = rows.Scan(&id, &sturdyRef, &serviceId, &payload, &authToken)
				if err != nil {
					t.Error(err)
				}
				if sturdyRef != tt.args.sturdyRef || serviceId != tt.args.serviceId || payload != tt.args.payload || authToken != tt.args.authToken {
					t.Errorf("Data not added correctly")
				}
			} else {
				t.Errorf("Data not added")
			}
		})
	}
}

func Test_getSturdyRef(t *testing.T) {

	db, err := setupdb(t, "test/test_get.db")
	if err != nil {
		return
	}
	defer db.Close()

	type args struct {
		db        *sql.DB
		sturdyRef string
	}
	tests := []struct {
		name          string
		args          args
		wantServiceId string
		wantPayload   string
		wantAuthToken string
		wantErr       bool
	}{
		{"Get Test", args{db, "test_sturdy_ref"}, "test_service_id", "test_payload", "test_auth_token", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := addSturdyRef(tt.args.db, tt.args.sturdyRef, tt.wantServiceId, tt.wantPayload, tt.wantAuthToken)
			if err != nil {
				t.Errorf("Error adding data")
			}

			got, got1, got2, err := getSturdyRef(tt.args.db, tt.args.sturdyRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSturdyRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantServiceId {
				t.Errorf("getSturdyRef() got = %v, want %v", got, tt.wantServiceId)
			}
			if got1 != tt.wantPayload {
				t.Errorf("getSturdyRef() got1 = %v, want %v", got1, tt.wantPayload)
			}
			if got2 != tt.wantAuthToken {
				t.Errorf("getSturdyRef() got2 = %v, want %v", got2, tt.wantAuthToken)
			}
		})
	}
}

func Test_deleteSturdyRef(t *testing.T) {
	db, err := setupdb(t, "test/test_delete.db")
	if err != nil {
		return
	}
	defer db.Close()

	type args struct {
		db        *sql.DB
		sturdyRef string
		serviceId string
		payload   string
		authToken string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Get delete test", args{db, "test_sturdy_ref", "test_service_id", "test_payload", "test_auth_token"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := addSturdyRef(tt.args.db, tt.args.sturdyRef, tt.args.serviceId, tt.args.payload, tt.args.authToken)
			if err != nil {
				t.Errorf("Error adding data")
			}

			if err := deleteSturdyRef(tt.args.db, tt.args.sturdyRef); (err != nil) != tt.wantErr {
				t.Errorf("deleteSturdyRef() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_listSturdyRefsByAuthToken(t *testing.T) {
	// check if db already exists
	// remove db if it exists from a previous test
	db, err := setupdb(t, "test/test_listBy.db")
	if err != nil {
		return
	}
	defer db.Close()

	// add data
	err = addSturdyRef(db, "test_sturdy_ref1", "test_service_id1", "test_payload1", "test_auth_token1")
	if err != nil {
		t.Errorf("Error adding data")
	}
	err = addSturdyRef(db, "test_sturdy_ref2", "test_service_id2", "test_payload2", "test_auth_token2")
	if err != nil {
		t.Errorf("Error adding data")
	}
	err = addSturdyRef(db, "test_sturdy_ref3", "test_service_id3", "test_payload3", "test_auth_token1")
	if err != nil {
		t.Errorf("Error adding data")
	}

	type args struct {
		db        *sql.DB
		authToken string
	}
	tests := []struct {
		name    string
		args    args
		want    []*sturdyRefStored
		wantErr bool
	}{
		{"List Test", args{db, "test_auth_token1"}, []*sturdyRefStored{
			{sturdyRef: "test_sturdy_ref1", serviceId: "test_service_id1", payload: "test_payload1"},
			{sturdyRef: "test_sturdy_ref3", serviceId: "test_service_id3", payload: "test_payload3"}},
			false},
		{"List Test", args{db, "test_auth_token2"}, []*sturdyRefStored{
			{sturdyRef: "test_sturdy_ref2", serviceId: "test_service_id2", payload: "test_payload2"}}, false},
		{"List Test", args{db, "test_auth_token3"}, []*sturdyRefStored{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := listSturdyRefsByAuthToken(tt.args.db, tt.args.authToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("listSturdyRefsByAuthToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("listSturdyRefsByAuthToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func setupdb(t *testing.T, dbPath string) (db *sql.DB, err error) {
	_, err = os.Stat(dbPath)
	if err == nil {
		err = os.Remove(dbPath)
		if err != nil {
			t.Error(err)
			t.Errorf("Error deleting db")
		}
	}
	db, err = createDB(dbPath)
	if err != nil {
		t.Errorf("Error creating db")
	}
	return db, err
}

func Test_listSturdyRefs(t *testing.T) {

	// check if db already exists
	// remove db if it exists from a previous test
	db, err := setupdb(t, "test/test_listBy.db")
	if err != nil {
		return
	}
	defer db.Close()

	// add data
	err = addSturdyRef(db, "test_sturdy_ref1", "test_service_id1", "test_payload1", "test_auth_token1")
	if err != nil {
		t.Errorf("Error adding data")
	}
	err = addSturdyRef(db, "test_sturdy_ref2", "test_service_id2", "test_payload2", "test_auth_token2")
	if err != nil {
		t.Errorf("Error adding data")
	}
	err = addSturdyRef(db, "test_sturdy_ref3", "test_service_id3", "test_payload3", "test_auth_token1")
	if err != nil {
		t.Errorf("Error adding data")
	}

	type args struct {
		db *sql.DB
	}
	tests := []struct {
		name    string
		args    args
		want    []*sturdyRefStored
		wantErr bool
	}{
		{"List Test", args{db}, []*sturdyRefStored{
			{sturdyRef: "test_sturdy_ref1", serviceId: "test_service_id1", payload: "test_payload1", authToken: "test_auth_token1"},
			{sturdyRef: "test_sturdy_ref2", serviceId: "test_service_id2", payload: "test_payload2", authToken: "test_auth_token2"},
			{sturdyRef: "test_sturdy_ref3", serviceId: "test_service_id3", payload: "test_payload3", authToken: "test_auth_token1"}},
			false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := listSturdyRefs(tt.args.db)
			if (err != nil) != tt.wantErr {
				t.Errorf("listSturdyRefs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("listSturdyRefs() = %v, want %v", got, tt.want)
			}
		})
	}
}
