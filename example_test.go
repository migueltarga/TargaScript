package targascript_test

import (
	"fmt"

	targascript "github.com/migueltarga/TargaScript"
)

// Users write a pricing rule; the Go program gives it the order and a
// function, and reads the result.
func Example() {
	rule := `
let subtotal = order.items.map(i => i.price * i.qty).sum()
match true {
    _ if order.coupon == "HALF" => subtotal / 2
    _ if subtotal > 100 => subtotal - shipping(order.country)
    _ => subtotal
}`

	script := targascript.New()
	script.Set("order", map[string]any{
		"items":   []map[string]any{{"price": 60, "qty": 2}, {"price": 5, "qty": 1}},
		"coupon":  "",
		"country": "BR",
	})
	script.Set("shipping", func(country string) int {
		if country == "BR" {
			return 15
		}
		return 30
	})

	total, err := script.Run(rule)
	fmt.Println(total, err)
	// Output: 110 <nil>
}
