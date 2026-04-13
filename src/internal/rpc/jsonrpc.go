package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

type jsonRPCReq struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type jsonRPCResp struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) call(ctx context.Context, method string, params []interface{}) (json.RawMessage, error) {
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}
	body, err := json.Marshal(jsonRPCReq{JSONRPC: "2.0", Method: method, Params: params, ID: 1})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out jsonRPCResp
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("jsonrpc decode: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("jsonrpc error %d: %s", out.Error.Code, out.Error.Message)
	}
	return out.Result, nil
}

// EthBlockNumber returns decimal block number from eth_blockNumber.
func (c *Client) EthBlockNumber(ctx context.Context) (uint64, error) {
	raw, err := c.call(ctx, "eth_blockNumber", []interface{}{})
	if err != nil {
		return 0, err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	return ParseHexUint64(s)
}

// ClientVersion returns web3_clientVersion result string.
func (c *Client) ClientVersion(ctx context.Context) (string, error) {
	raw, err := c.call(ctx, "web3_clientVersion", []interface{}{})
	if err != nil {
		return "", err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", err
	}
	return s, nil
}

// NetVersion returns net_version result string.
func (c *Client) NetVersion(ctx context.Context) (string, error) {
	raw, err := c.call(ctx, "net_version", []interface{}{})
	if err != nil {
		return "", err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", err
	}
	return s, nil
}

func ParseHexUint64(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty hex")
	}
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return strconv.ParseUint(s[2:], 16, 64)
	}
	return strconv.ParseUint(s, 10, 64)
}

// ParseHexBigInt parses a hex quantity (e.g. eth_getBalance result).
func ParseHexBigInt(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty hex")
	}
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if s == "" {
		return big.NewInt(0), nil
	}
	b := new(big.Int)
	if _, ok := b.SetString(s, 16); !ok {
		return nil, fmt.Errorf("invalid hex int: %q", s)
	}
	return b, nil
}

// EthGetBalance returns account balance in wei at blockTag (e.g. "latest").
func (c *Client) EthGetBalance(ctx context.Context, addressHex, blockTag string) (*big.Int, error) {
	raw, err := c.call(ctx, "eth_getBalance", []interface{}{addressHex, blockTag})
	if err != nil {
		return nil, err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return ParseHexBigInt(s)
}
