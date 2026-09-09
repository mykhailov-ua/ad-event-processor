package export

func paginateCHExport[T any](
	pageSize int,
	maxRows int,
	query func(offset, limit int) ([]T, int64, error),
	write func(T) error,
) error {
	if pageSize <= 0 {
		pageSize = 1000
	}
	written := 0
	for offset := 0; ; offset += pageSize {
		limit := pageSize
		if maxRows > 0 && written+limit > maxRows {
			limit = maxRows - written
			if limit <= 0 {
				return nil
			}
		}
		rows, total, err := query(offset, limit)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if err := write(row); err != nil {
				return err
			}
			written++
			if maxRows > 0 && written >= maxRows {
				return nil
			}
		}
		if int64(offset+len(rows)) >= total || len(rows) == 0 {
			return nil
		}
	}
}
