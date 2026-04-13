package coingecko

import (
	"math"
	"testing"
)

func TestParseSimplePriceResponse(t *testing.T) {
	const body = `{"monad":{"usd":0.042}}`
	v, err := ParseSimplePriceResponse([]byte(body), "monad")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(v-0.042) > 1e-9 {
		t.Fatalf("got %v want 0.042", v)
	}
	_, err = ParseSimplePriceResponse([]byte(`{"other":{"usd":1}}`), "monad")
	if err == nil {
		t.Fatal("expected error for missing coin")
	}
}
