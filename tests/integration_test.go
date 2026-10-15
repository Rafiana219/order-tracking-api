package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

type apiError struct {
	Error string `json:"error"`
}

type createOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type createOrderAPIResponse struct {
	Data createOrderResponse `json:"data"`
}

func apiBaseURL() string {
	if url := os.Getenv("API_BASE_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

func sendJSON(
	t *testing.T,
	method string,
	path string,
	idempotencyKey string,
	body any,
) (*http.Response, []byte) {
	t.Helper()

	var requestBody []byte
	if body != nil {
		var err error
		requestBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
	}

	req, err := http.NewRequest(
		method,
		apiBaseURL()+path,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}

	t.Cleanup(func() {
		resp.Body.Close()
	})

	var responseBody bytes.Buffer
	if _, err := responseBody.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read response: %v", err)
	}

	return resp, responseBody.Bytes()
}

func createOrderBody(productID string, quantity int) map[string]any {
	return map[string]any{
		"customer_id": "integration-customer",
		"items": []map[string]any{
			{
				"product_id": productID,
				"quantity":   quantity,
			},
		},
	}
}

func TestCreateOrder(t *testing.T) {
	key := fmt.Sprintf("integration-create-%d", time.Now().UnixNano())

	resp, body := sendJSON(
		t,
		http.MethodPost,
		"/api/orders/",
		key,
		createOrderBody("product-test", 1),
	)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d; response: %s",
			resp.StatusCode,
			http.StatusCreated,
			body,
		)
	}

	// var order createOrderResponse
	// if err := json.Unmarshal(body, &order); err != nil {
	// 	t.Fatalf("decode response: %v; body: %s", err, body)
	// }

	// if order.ID == "" {
	// 	t.Fatal("expected order ID, got empty string")
	// }

	// t.Logf("Create Order response: %s", string(body))

	var result createOrderAPIResponse

	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode response: %v; body: %s", err, body)
	}

	if result.Data.ID == "" {
		t.Fatalf("expected order ID in data, got: %s", body)
	}

	if result.Data.Status != "PENDING_PAYMENT" {
		t.Errorf(
			"status = %q, want %q",
			result.Data.Status,
			"PENDING_PAYMENT",
		)
	}
}

func TestIdempotencyConflict(t *testing.T) {
	key := fmt.Sprintf("integration-conflict-%d", time.Now().UnixNano())

	firstResp, firstBody := sendJSON(
		t,
		http.MethodPost,
		"/api/orders/",
		key,
		createOrderBody("product-A", 1),
	)

	if firstResp.StatusCode != http.StatusCreated {
		t.Fatalf("first request status = %d; response: %s",
			firstResp.StatusCode, firstBody)
	}

	secondResp, secondBody := sendJSON(
		t,
		http.MethodPost,
		"/api/orders/",
		key,
		createOrderBody("product-B", 2),
	)

	if secondResp.StatusCode != http.StatusConflict {
		t.Fatalf("second request status = %d, want %d; response: %s",
			secondResp.StatusCode,
			http.StatusConflict,
			secondBody,
		)
	}
}

func TestConfirmPaymentAfterCompletion(t *testing.T) {
	orderID := os.Getenv("COMPLETED_ORDER_ID")
	if orderID == "" {
		t.Skip("set COMPLETED_ORDER_ID to an existing COMPLETED order")
	}

	resp, body := sendJSON(
		t,
		http.MethodPost,
		"/api/orders/"+orderID+"/confirm-payment",
		"",
		nil,
	)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want %d; response: %s",
			resp.StatusCode,
			http.StatusConflict,
			body,
		)
	}
}
