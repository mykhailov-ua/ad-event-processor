package billingadmin

import (
	"fmt"

	"ad-event-processor/pkg/money"
)

func ParseMoneyMicro(micro *int64, legacy float64, hasLegacy bool, field string) (int64, error) {
	if micro != nil {
		if *micro < 0 {
			return 0, errValidation(fmt.Sprintf("invalid %s", field))
		}
		return *micro, nil
	}
	if hasLegacy {
		v, err := money.LegacyFloatToMicro(legacy)
		if err != nil {
			return 0, errValidation(fmt.Sprintf("invalid %s", field))
		}
		return v, nil
	}
	return 0, nil
}

func ParseBudgetMicro(micro *int64, legacy float64, hasLegacy bool) (int64, error) {
	if micro != nil {
		if *micro <= 0 {
			return 0, errValidation("budget must be positive")
		}
		return *micro, nil
	}
	if hasLegacy {
		v, err := money.LegacyFloatToMicro(legacy)
		if err != nil || v <= 0 {
			return 0, errValidation("budget must be positive")
		}
		return v, nil
	}
	return 0, errValidation("budget is required")
}

func OptionalBudgetMicro(micro *int64, legacy *float64) (*int64, error) {
	if micro != nil {
		if *micro <= 0 {
			return nil, errValidation("budget must be positive")
		}
		v := *micro
		return &v, nil
	}
	if legacy != nil {
		v, err := money.LegacyFloatToMicro(*legacy)
		if err != nil || v <= 0 {
			return nil, errValidation("budget must be positive")
		}
		return &v, nil
	}
	return nil, nil
}

func FormatMicro(m int64) string {
	return money.FormatFixed2(m)
}
