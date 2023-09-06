package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"treasure/database"
)

var urlTreasure = "https://www.treasury.gov/ofac/downloads/sdn.xml"
var parsing = 0

func main() {

	host := os.Getenv("DD_DB_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	database.DBCon, _ = sql.Open("postgres", "postgres://user:password@"+host+"/treasure?sslmode=disable")
	defer func(DBCon *sql.DB) {
		err := DBCon.Close()
		if err != nil {

		}
	}(database.DBCon)

	handleRequests()
}

func welcome(w http.ResponseWriter, _ *http.Request) {
	_, err := fmt.Fprintf(w, "Привет, нравится?")
	if err != nil {
		return
	}
}

func getResultFromSql(rows *sql.Rows) []map[string]interface{} {
	columns, _ := rows.Columns()

	var allMaps []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		err := rows.Scan(pointers...)
		if err != nil {
			log.Fatal(err)
		}
		resultMap := make(map[string]interface{})
		for i, val := range values {
			var v interface{}
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			resultMap[columns[i]] = v
		}
		allMaps = append(allMaps, resultMap)
	}

	return allMaps
}

func returnUpdate(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	parsing = 1

	resp, err := http.Get(urlTreasure)
	if err != nil {
		log.Fatalln(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != 200 {
		err := json.NewEncoder(w).Encode(APIAnswer{
			Result: false,
			Info:   "service unavailable",
			Code:   resp.StatusCode,
		})
		if err != nil {
			return
		}

		return
	}

	var sdnList SDNList
	if err = xml.NewDecoder(resp.Body).Decode(&sdnList); err != nil {
		log.Fatalln(err)
	}

	res, err := database.DBCon.Query("TRUNCATE TABLE sdn")
	defer func(res *sql.Rows) {
		err := res.Close()
		if err != nil {

		}
	}(res)

	if err != nil {
		log.Fatal(err)
	}

	for _, url := range sdnList.Entries {
		if url.SDNType == "Individual" {
			res, err := database.DBCon.Query("INSERT INTO sdn (uid, first_name, last_name) VALUES ($1, $2, $3)", url.UID, url.FirstName, url.LastName)
			err = res.Close()
			if err != nil {
				return
			}

			if err != nil {
				log.Fatal(err)
			}
		}
	}

	parsing = 0

	err = json.NewEncoder(w).Encode(APIAnswer{
		Result: true,
		Info:   "",
		Code:   resp.StatusCode,
	})
	if err != nil {
		return
	}
}

func returnState(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if parsing == 0 {
		res, err := database.DBCon.Query("SELECT uid FROM sdn LIMIT 1")
		defer func(res *sql.Rows) {
			err := res.Close()
			if err != nil {

			}
		}(res)

		if err != nil {
			log.Fatal(err)
		}
		if len(getResultFromSql(res)) == 1 {
			err := json.NewEncoder(w).Encode(APIAnswer{
				Result: true,
				Info:   "ok",
			})
			if err != nil {
				return
			}
		} else {
			err := json.NewEncoder(w).Encode(APIAnswer{
				Result: false,
				Info:   "empty",
			})
			if err != nil {
				return
			}
		}
	} else {
		err := json.NewEncoder(w).Encode(APIAnswer{
			Result: false,
			Info:   "updating",
		})
		if err != nil {
			return
		}
	}

}

func returnGetNames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	name := r.URL.Query().Get("name")
	typeX := r.URL.Query().Get("type")

	query := ""

	if strings.ToLower(typeX) == "strong" {
		query = "SELECT * FROM sdn WHERE first_name = $1 OR last_name = $1"
	} else {
		name = strings.ReplaceAll(name, " ", " or ")
		query = "SELECT * FROM sdn WHERE (to_tsvector('simple', first_name) || to_tsvector('simple', last_name)) @@ websearch_to_tsquery('simple',$1);"
	}

	res, err := database.DBCon.Query(query, name)
	defer func(res *sql.Rows) {
		err := res.Close()
		if err != nil {

		}
	}(res)

	if err != nil {
		log.Fatal(err)
	}

	var sdn SDNEntry
	var sdns []SDNEntry
	for res.Next() {

		err := res.Scan(&sdn.UID, &sdn.FirstName, &sdn.LastName)

		if err != nil {
			log.Fatal(err)
		}

		sdns = append(sdns, sdn)
	}

	if sdns == nil {
		err := json.NewEncoder(w).Encode(APIAnswer{
			Result: false,
			Info:   "no data",
		})
		if err != nil {
			return
		}
		return
	}
	err = json.NewEncoder(w).Encode(sdns)
	if err != nil {
		return
	}

}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.Use(mux.CORSMethodMiddleware(myRouter))

	myRouter.HandleFunc("/", welcome)

	myRouter.HandleFunc("/update", returnUpdate)
	myRouter.HandleFunc("/state", returnState)
	myRouter.HandleFunc("/get_names", returnGetNames)

	log.Fatal(http.ListenAndServe(":8080", handlers.CompressHandler(myRouter)))
}
