package postback

import (
	"testing"
)

func FuzzPostbackURLExpand(f *testing.F) {
	f.Add("https://aff.example.com/pb?click_id={click_id}&sub1={sub1}&payout={payout}")
	f.Add("{click_id}{sub30}{unknown}")
	f.Fuzz(func(t *testing.T, tpl string) {
		if len(tpl) > MaxRenderedURLLen*2 {
			tpl = tpl[:MaxRenderedURLLen*2]
		}
		mt := ParseTemplate(tpl)
		var scratch [MaxRenderedURLLen]byte
		ctx := EventContext{
			ClickID: "fuzz-click",
			Payout:  "1.00",
			TxID:    "tx",
		}
		ctx.SubIDs[0] = "sub"
		_ = mt.RenderStack(&ctx, &scratch)
	})
}
