package validation

import (
	"fmt"
	"math"

	validatorv10 "github.com/go-playground/validator/v10"
)

// tolerance for floating point comparison (in cents)
// const amountTolerance = 0.01

// new returns a configured validator with custom struct-level validation registered.
func New() *validatorv10.Validate {
	v := validatorv10.New()

	// register struct-level validation for CreateOrderRequest to ensure
	// the provided Amount matches the sum of (price * quantity) of items.
	v.RegisterStructValidation(createOrderStructValidation, CreateOrderRequest{})

	return v
}

// createOrderStructValidation verifies the aggregated total of items equals Amount (within cents)
func createOrderStructValidation(sl validatorv10.StructLevel) {
	req := sl.Current().Interface().(CreateOrderRequest)

	var sum float64
	for _, it := range req.Items {
		sum += float64(it.Quantity) * it.Price
	}

	sumCents := int(math.Round(sum * 100))
	amountCents := int(math.Round(req.Amount * 100))
	if sumCents != amountCents {
		sl.ReportError(req.Amount, "amount", "Amount", "amount_match_items", fmt.Sprintf("items sum %.2f != amount %.2f", sum, req.Amount))
	}

	// // use a tolerance to avoid float rounding issues
	// if math.Abs(sum-req.Amount) >= amountTolerance {
	// 	sl.ReportError(req.Amount, "amount", "Amount", "amount_match_items", fmt.Sprintf("items sum %.2f != amount %.2f", sum, req.Amount))
	// }
}
