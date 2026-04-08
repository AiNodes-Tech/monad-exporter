package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// EthCall executes eth_call against the given contract address with calldata (no 0x prefix required in data).
func (c *Client) EthCall(ctx context.Context, to string, data []byte) ([]byte, error) {
	to = strings.TrimSpace(to)
	if !strings.HasPrefix(to, "0x") && !strings.HasPrefix(to, "0X") {
		to = "0x" + to
	}
	dataHex := "0x" + hex.EncodeToString(data)
	callObj := map[string]interface{}{
		"to":   to,
		"data": dataHex,
	}
	raw, err := c.call(ctx, "eth_call", []interface{}{callObj, "latest"})
	if err != nil {
		return nil, err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("eth_call result: %w", err)
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "0x" {
		return nil, nil
	}
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	return hex.DecodeString(strings.TrimPrefix(s, "0x"))
}
