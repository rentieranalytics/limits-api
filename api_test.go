package limitsapi

import (
	"encoding/json"
	"testing"
)

// The JSON shape is the contract between services; these tests pin it.

func TestRequestsAreFlat(t *testing.T) {
	b, err := json.Marshal(LinkRequest{
		OperationID: "op-1",
		Key:         Key{Resource: ResourceValuation, Subject: "widget"},
		Parent:      "account:1",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"operation_id":"op-1","resource":"valuation","subject":"widget","parent":"account:1"}`
	if string(b) != want {
		t.Fatalf("got  %s\nwant %s", b, want)
	}
}

func TestReleaseResponse(t *testing.T) {
	b, err := json.Marshal(ReleaseResponse{Reason: ReleaseExpired})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"returned":false,"reason":"expired"}`; string(b) != want {
		t.Fatalf("got  %s\nwant %s", b, want)
	}
}

func TestLimitOmitsUnsetFields(t *testing.T) {
	b, err := json.Marshal(Limit{Key: Key{Resource: "valuation", Subject: "w"}, State: StateUnlimited, Period: "2026-10"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"resource":"valuation","subject":"w","state":"unlimited","monthly":0,"next_monthly":0,"used":0,"period":"2026-10","reset_at":"0001-01-01T00:00:00Z"}`
	if string(b) != want {
		t.Fatalf("got  %s\nwant %s", b, want)
	}
}

func TestResponseEnvelope(t *testing.T) {
	var ok Response[ConsumeResponse]
	if err := json.Unmarshal([]byte(`{"data":{"granted":true,"replayed":true}}`), &ok); err != nil {
		t.Fatal(err)
	}
	if ok.Error != nil || !ok.Data.Granted || !ok.Data.Replayed {
		t.Fatalf("data: %+v", ok)
	}

	var failed Response[Limit]
	if err := json.Unmarshal([]byte(`{"error":{"code":"internal","message":"internal error"}}`), &failed); err != nil {
		t.Fatal(err)
	}
	if failed.Data != nil || failed.Error.Code != CodeInternal {
		t.Fatalf("error: %+v", failed)
	}
}

func TestOnlyInternalErrorsAreTemporary(t *testing.T) {
	for _, code := range []string{
		CodeInvalidRequest, CodeInvalidAmount, CodeNotConfigured,
		CodeAlreadyConfigured, CodeUnlimited, CodeOperationConflict,
	} {
		if (&Error{Code: code}).Temporary() {
			t.Errorf("%s must be permanent", code)
		}
	}
	if !(&Error{Code: CodeInternal}).Temporary() {
		t.Error("internal must be temporary")
	}
}
