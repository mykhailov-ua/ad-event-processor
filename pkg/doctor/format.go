package doctor

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

func WriteReport(w io.Writer, rep Report) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "PROBE\tSTATUS\tDETAIL\tMS")
	for _, r := range rep.Results {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%d\n", r.Name, r.Status, sanitizeDetail(r.Detail), r.Latency)
	}
	_ = tw.Flush()
}

func WriteChecklist(w io.Writer, rows []ChecklistRow) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "MVSS\tSTATUS\tDETAIL")
	for _, row := range rows {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", row.ID, row.Status, sanitizeDetail(row.Detail))
	}
	_ = tw.Flush()
}

func sanitizeDetail(s string) string {
	return strings.ReplaceAll(s, "\t", " ")
}
