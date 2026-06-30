package migrate

import (
	"embed"
	"database/sql"
	"io/fs"
	"sort"
)

//go:embed migrations/*.sql
var MigrationsFS embed.FS

func Run(db *sql.DB) error {
	files, err := fs.Glob(MigrationsFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		sqlBytes, err := MigrationsFS.ReadFile(file)
		if err != nil {
			return err
		}
		for _, stmt := range splitSQL(string(sqlBytes)) {
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return err
			}
		}
	}
	return nil
}

func splitSQL(sql string) []string {
	var stmts []string
	current := ""
	inString := false
	stringChar := byte(0)
	for i := 0; i < len(sql); i++ {
		c := sql[i]
		if !inString {
			if c == '\'' || c == '"' {
				inString = true
				stringChar = c
			}
			if c == ';' {
				stmts = append(stmts, current)
				current = ""
				continue
			}
		} else {
			if c == stringChar {
				inString = false
			}
		}
		current += string(c)
	}
	return stmts
}