package order

type TaxType string

const (
	TaxTypePPN  TaxType = "ppn"
	TaxTypePB1  TaxType = "pb1"
	TaxTypeNone TaxType = "none"
)

type TaxConfig struct {
	Type TaxType
	Rate int64 // basis points: PPN=1100 (11%), PB1=1000 (10%), none=0
}

// Calculate returns (taxAmount, total). PB1 is inclusive — tax reported but not added to total.
func (c TaxConfig) Calculate(subtotal int64) (taxAmount, total int64) {
	if c.Rate <= 0 || c.Type == TaxTypeNone {
		return 0, subtotal
	}
	taxAmount = subtotal * c.Rate / 10000
	if c.Type == TaxTypePB1 {
		return taxAmount, subtotal
	}
	return taxAmount, subtotal + taxAmount
}
