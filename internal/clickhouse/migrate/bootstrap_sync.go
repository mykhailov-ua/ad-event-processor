package migrate

import (
	"io/fs"
	"regexp"
	"strings"
)

const bootstrapMigrationFile = "00000_bootstrap_tables.sql"

func BootstrapMigrationFile() string {
	return bootstrapMigrationFile
}

var sqlWhitespaceRE = regexp.MustCompile(`\s+`)

func BootstrapMigrationSQL() (string, error) {
	body, err := fs.ReadFile(ClickHouseMigrationFS(), bootstrapMigrationFile)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func NormalizeBootstrapDDL(sql string) string {
	var b strings.Builder
	for _, line := range strings.Split(sql, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "--") {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(trim), "USE ") {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(strings.TrimSpace(sqlWhitespaceRE.ReplaceAllString(trim, " ")))
	}
	return b.String()
}

func InitSQLTableSection(initSQL string) string {
	lines := strings.Split(initSQL, "\n")
	var out []string
	pastHeader := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if !pastHeader {
			if strings.HasPrefix(trim, "--") || trim == "" {
				continue
			}
			if strings.HasPrefix(strings.ToUpper(trim), "CREATE DATABASE") {
				continue
			}
			if strings.HasPrefix(strings.ToUpper(trim), "USE ") {
				pastHeader = true
				continue
			}
		}
		if pastHeader {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
