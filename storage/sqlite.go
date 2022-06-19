package storage

import (
	"database/sql"
	"encoding/json"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"time"
)

type SqliteStorage struct {
	db *sql.DB
}

func NewSqliteStorage() *SqliteStorage {
	db, err := sql.Open("sqlite", "st.db")
	if err != nil {
		log.Fatal(err)
	}

	prepareDatabase(db)

	st := &SqliteStorage{
		db: db,
	}
	go st.StartAutoDeletes()
	return st
}

func (st *SqliteStorage) PushRequestToInbox(name string, request http.Request) {
	if st.db == nil {
		log.Fatal("No DB connection to push request to.")
	}

	queryStr, err := json.Marshal(request.URL.Query())
	if err != nil {
		log.Fatal("Error serializing query params", err)
	}

	headerStr, err := json.Marshal(request.Header)
	if err != nil {
		log.Fatal("Error serializing headers", err)
	}

	result, err := st.db.Exec(
		`
			insert into requests (inbox_name, protocol, scheme, host, path, method, params, headers, fragment)
			values (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
		name,
		request.Proto,
		request.URL.Scheme,
		request.Host,
		request.URL.Path,
		request.Method,
		queryStr,
		headerStr,
		request.URL.Fragment,
	)

	if err != nil {
		log.Fatal("Error creating statement for inserting request", err)
	}

	log.Printf("Inserted request %v.", result)
}

func (st *SqliteStorage) GetFromInbox(name string) []Entry {
	if st.db == nil {
		return nil
	}

	entries := []Entry{}

	rows, err := st.db.Query("select protocol, scheme, host, path, method, params, headers, fragment, pushedAt from requests where inbox_name == ?", name)
	if err != nil {
		log.Fatal("Error selecting requests", err)
	}

	for rows.Next() {
		var protocol string
		var scheme string
		var host string
		var path string
		var method string
		var paramsStr string
		var headersStr string
		var fragment string
		var pushedAtStr string
		rows.Scan(
			&protocol,
			&scheme,
			&host,
			&path,
			&method,
			&paramsStr,
			&headersStr,
			&fragment,
			&pushedAtStr,
		)
		var params map[string][]string
		if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
			log.Print("Error deserializing params", err)
		}
		var headers map[string][]string
		if err := json.Unmarshal([]byte(headersStr), &headers); err != nil {
			log.Print("Error deserializing headers", err)
		}
		pushedAt, err := time.Parse(time.RFC3339, pushedAtStr)
		if err != nil {
			log.Print("Error parsing time", err)
		}
		entries = append(entries, Entry{
			Protocol: protocol,
			Scheme:   scheme,
			Host:     host,
			Path:     path,
			Method:   method,
			Params:   params,
			Headers:  headers,
			Fragment: fragment,
			PushedAt: pushedAt,
		})
	}

	return entries
}

func (st *SqliteStorage) StartAutoDeletes() {
	st.DoAutoDelete()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		<-ticker.C
		st.DoAutoDelete()
	}
}

func (st *SqliteStorage) DoAutoDelete() {
	_, err := st.db.Exec(
		"delete from requests where pushedAt < ?",
		time.Now().UTC().Add(10*time.Minute),
	)
	if err != nil {
		log.Fatal("Error deleting old requests", err)
	}
}

func prepareDatabase(db *sql.DB) {
	var version int
	row := db.QueryRow("select n from version limit 1")
	if err := row.Err(); err != nil {
		if err.Error() != "no such table: version" {
			log.Fatal(err)
		}

	} else if err = row.Scan(&version); err != nil {
		log.Fatal(err)

	}

	log.Printf("Found DB Version %v.\n", version)

	if version < 1 {
		_, err := db.Exec("create table version (n integer not null); insert into version values (1);")
		if err != nil {
			log.Fatal(err)
		}
	}

	if version < 2 {
		_, err := db.Exec(`
			create table requests (
				id integer not null primary key,
				inbox_name varchar(20),
				protocol varchar(10),
				scheme varchar(10),
				host varchar(90),
				path varchar(90),
				method varchar(10),
				params varchar(300),
				headers varchar(300),
				fragment varchar(90),
				pushedAt timestamp default current_timestamp
			);
			update version set n = 2;
		`)
		if err != nil {
			log.Fatal(err)
		}
	}
}
