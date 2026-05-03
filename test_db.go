package main

import (
    "context"
    "fmt"
    "log"
    "database/sql"
    _ "github.com/lib/pq"
)

func main() {
    db, err := sql.Open("postgres", "postgres://postgres:password@localhost:5432/rds?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    
    query := `SELECT COALESCE(MAX(node_port), 999) + 1 FROM databases WHERE node_port >= 1000`
var port int
err = db.QueryRowContext(context.Background(), query).Scan(&port)
    fmt.Printf("GetNextNodePort returned: %v (err: %v)\n", port, err)

    query2 := `SELECT node_port FROM databases`
    rows, _ := db.Query(query2)
    for rows.Next() {
        var p int
        rows.Scan(&p)
        fmt.Printf("DB has port: %d\n", p)
    }
}
