package main

import (
	"database/sql"
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
			got, err := createDB(tt.args.dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("createDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				got.Close()
			}
		})
	}
}

func Test_addSturdyRef(t *testing.T) {
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := addSturdyRef(tt.args.db, tt.args.sturdyRef, tt.args.serviceId, tt.args.payload, tt.args.authToken); (err != nil) != tt.wantErr {
				t.Errorf("addSturdyRef() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getSturdyRef(t *testing.T) {
	type args struct {
		db        *sql.DB
		sturdyRef string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		want2   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, err := getSturdyRef(tt.args.db, tt.args.sturdyRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSturdyRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getSturdyRef() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getSturdyRef() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("getSturdyRef() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func Test_deleteSturdyRef(t *testing.T) {
	type args struct {
		db        *sql.DB
		sturdyRef string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := deleteSturdyRef(tt.args.db, tt.args.sturdyRef); (err != nil) != tt.wantErr {
				t.Errorf("deleteSturdyRef() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_listSturdyRefsByAuthToken(t *testing.T) {
	type args struct {
		db        *sql.DB
		authToken string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
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

func Test_listSturdyRefs(t *testing.T) {
	type args struct {
		db *sql.DB
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
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
