package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

const xlsxSheetName = "Sheet1"

func convertCSVFileToXLSX(csvPath, xlsxPath string) (int, error) {
	in, err := os.Open(csvPath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = in.Close() }()

	reader := csv.NewReader(in)
	reader.ReuseRecord = true
	// Export CSV mixes 2-column meta preamble rows with wider report headers/data.
	reader.FieldsPerRecord = -1

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err := f.SetSheetName("Sheet1", xlsxSheetName); err != nil {
		return 0, err
	}
	sw, err := f.NewStreamWriter(xlsxSheetName)
	if err != nil {
		return 0, err
	}

	rowNum := 0
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, fmt.Errorf("read csv row %d: %w", rowNum+1, readErr)
		}
		if len(record) > 0 && strings.HasPrefix(record[0], "#") {
			continue
		}
		rowNum++
		cells := make([]any, len(record))
		for i, value := range record {
			cells[i] = value
		}
		if err := sw.SetRow(fmt.Sprintf("A%d", rowNum), cells); err != nil {
			return 0, fmt.Errorf("write xlsx row %d: %w", rowNum, err)
		}
	}
	if err := sw.Flush(); err != nil {
		return 0, err
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(xlsxPath, buf.Bytes(), 0o600); err != nil {
		return 0, err
	}
	return rowNum, nil
}
