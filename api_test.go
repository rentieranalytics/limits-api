package limitsapi

import (
	"encoding/json"
	"testing"
	"time"
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

func TestAddUnitsWithValidity(t *testing.T) {
	until := time.Date(2027, time.October, 19, 0, 0, 0, 0, time.UTC)
	b, err := json.Marshal(AddUnitsRequest{OperationID: "op", Key: Key{Resource: "valuation", Subject: "s"}, Units: 25, ValidUntil: &until})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"operation_id":"op","resource":"valuation","subject":"s","units":25,"valid_until":"2027-10-19T00:00:00Z"}`
	if string(b) != want {
		t.Fatalf("got  %s\nwant %s", b, want)
	}
	b, _ = json.Marshal(AddUnitsRequest{OperationID: "op", Key: Key{Resource: "valuation", Subject: "s"}, Units: 5})
	if want := `{"operation_id":"op","resource":"valuation","subject":"s","units":5}`; string(b) != want {
		t.Fatalf("without validity: %s", b)
	}
}

func TestPeriodAnchor(t *testing.T) {
	anchor := 19
	b, _ := json.Marshal(SetMonthlyRequest{OperationID: "op", Key: Key{Resource: "valuation", Subject: "s"}, Monthly: 75, PeriodAnchor: &anchor})
	if want := `{"operation_id":"op","resource":"valuation","subject":"s","monthly":75,"add_current":0,"period_anchor":19}`; string(b) != want {
		t.Fatalf("set_monthly: %s", b)
	}
	b, _ = json.Marshal(SetMonthlyRequest{OperationID: "op", Key: Key{Resource: "valuation", Subject: "s"}, Monthly: 75})
	if want := `{"operation_id":"op","resource":"valuation","subject":"s","monthly":75,"add_current":0}`; string(b) != want {
		t.Fatalf("without anchor: %s", b)
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
		CodeAlreadyConfigured, CodeUnlimited, CodeOperationConflict, CodeGrantNotFound,
	} {
		if (&Error{Code: code}).Temporary() {
			t.Errorf("%s must be permanent", code)
		}
	}
	if !(&Error{Code: CodeInternal}).Temporary() {
		t.Error("internal must be temporary")
	}
}
