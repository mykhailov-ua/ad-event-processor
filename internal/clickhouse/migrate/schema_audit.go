package migrate

import (
	"regexp"
	"strings"
)

var ForbiddenCHColumns = []string{
	"ip_address",
	"user_agent",
	"user_id",
}

var createTableRE = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:ad_event_processor\.)?(\w+)`)

func AuditSchemaDDL(files map[string]string) []SchemaAuditViolation {
	var violations []SchemaAuditViolation
	for path, body := range files {
		violations = append(violations, auditFile(path, body)...)
	}
	return violations
}

type SchemaAuditViolation struct {
	File   string
	Table  string
	Column string
}

func auditFile(path, body string) []SchemaAuditViolation {
	var out []SchemaAuditViolation
	currentTable := ""
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if m := createTableRE.FindStringSubmatch(trim); len(m) == 2 {
			currentTable = m[1]
			continue
		}
		if currentTable == "" {
			continue
		}
		if trim == ")" || strings.HasPrefix(trim, ") ENGINE") {
			currentTable = ""
			continue
		}
		lower := strings.ToLower(trim)
		for _, col := range ForbiddenCHColumns {
			if strings.HasPrefix(lower, col+" ") || strings.HasPrefix(lower, col+"\t") {
				out = append(out, SchemaAuditViolation{
					File:   path,
					Table:  currentTable,
					Column: col,
				})
			}
		}
	}
	return out
}
