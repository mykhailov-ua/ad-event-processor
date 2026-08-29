package clickhouse

type invalidQueryError string

func (e invalidQueryError) Error() string { return string(e) }

func errInvalidQuery(msg string) error { return invalidQueryError(msg) }
