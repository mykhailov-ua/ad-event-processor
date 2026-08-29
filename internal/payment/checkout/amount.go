package checkout

import "fmt"

const stripeCentsPerMicro = 10_000

func StripeAmountToMicro(stripeAmount int64) int64 {
	return stripeAmount * stripeCentsPerMicro
}

func MicroToStripeAmount(amountMicro int64) (int64, error) {
	if amountMicro%stripeCentsPerMicro != 0 {
		return 0, fmt.Errorf("amount_micro %d is not aligned to Stripe cent granularity", amountMicro)
	}
	return amountMicro / stripeCentsPerMicro, nil
}
