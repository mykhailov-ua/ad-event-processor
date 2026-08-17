// Package money implements money support for BidShard.
package money

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const MicroUnit = int64(1_000_000)

var ErrInvalidAmount = errors.New("invalid money amount")

func ParseDecimal(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = strings.TrimPrefix(s, "-")
	}
	if s == "" {
		return 0, ErrInvalidAmount
	}

	var whole, frac int64
	if strings.Contains(s, ".") {
		parts := strings.SplitN(s, ".", 2)
		if _, err := fmt.Sscanf(parts[0], "%d", &whole); err != nil {
			return 0, ErrInvalidAmount
		}
		fracStr := parts[1]
		if len(fracStr) > 6 {
			fracStr = fracStr[:6]
		}
		for len(fracStr) < 6 {
			fracStr += "0"
		}
		if _, err := fmt.Sscanf(fracStr, "%d", &frac); err != nil {
			return 0, ErrInvalidAmount
		}
	} else {
		if _, err := fmt.Sscanf(s, "%d", &whole); err != nil {
			return 0, ErrInvalidAmount
		}
	}
	if whole < 0 || frac < 0 {
		return 0, ErrInvalidAmount
	}
	out := whole*MicroUnit + frac
	if neg {
		out = -out
	}
	return out, nil
}

func LegacyFloatToMicro(v float64) (int64, error) {
	if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, ErrInvalidAmount
	}
	return ParseDecimal(strconv.FormatFloat(v, 'f', 6, 64))
}

func JSONAmountToMicro(v float64) (int64, error) {
	return LegacyFloatToMicro(v)
}

func PercentBps(amountMicro, bps int64) int64 {
	if amountMicro <= 0 || bps <= 0 {
		return 0
	}
	return amountMicro * bps / 10000
}

func PercentFromFloat(amountMicro int64, percent float64) int64 {
	if amountMicro <= 0 || percent <= 0 {
		return 0
	}
	return PercentHundredths(amountMicro, int64(math.Round(percent*100)))
}

func PercentHundredths(amountMicro, hundredths int64) int64 {
	if amountMicro <= 0 || hundredths <= 0 {
		return 0
	}
	return amountMicro * hundredths / 10000
}

func ScalePPM(amountMicro, ppm int64) int64 {
	if amountMicro <= 0 || ppm <= 0 {
		return 0
	}
	return amountMicro * ppm / 1_000_000
}

func MulMicro(perUnitMicro, count int64) int64 {
	if perUnitMicro <= 0 || count <= 0 {
		return 0
	}
	return perUnitMicro * count
}

func FormatFixed2(micro int64) string {
	neg := micro < 0
	if neg {
		micro = -micro
	}
	whole := micro / MicroUnit
	frac := (micro % MicroUnit) / 10_000
	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%s%d.%02d", sign, whole, frac)
}

func FormatDecimal(micro int64) string {
	neg := micro < 0
	if neg {
		micro = -micro
	}
	whole := micro / MicroUnit
	frac := micro % MicroUnit
	if frac == 0 {
		if neg {
			return "-" + strconv.FormatInt(whole, 10)
		}
		return strconv.FormatInt(whole, 10)
	}
	fracStr := strconv.FormatInt(frac, 10)
	for len(fracStr) < 6 {
		fracStr = "0" + fracStr
	}
	fracStr = strings.TrimRight(fracStr, "0")
	sign := ""
	if neg {
		sign = "-"
	}
	return sign + strconv.FormatInt(whole, 10) + "." + fracStr
}

func APIValueFloat(micro int64) float64 {
	return float64(micro) / float64(MicroUnit)
}
