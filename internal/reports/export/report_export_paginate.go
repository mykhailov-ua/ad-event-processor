package export

func paginateCHExport[T any](
	pageSize int,
	query func(offset, limit int) ([]T, int64, error),
	write func(T) error,
) error {
	for offset := 0; ; offset += pageSize {
		rows, total, err := query(offset, pageSize)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if err := write(row); err != nil {
				return err
			}
		}
		if int64(offset+len(rows)) >= total || len(rows) == 0 {
			return nil
		}
	}
}
