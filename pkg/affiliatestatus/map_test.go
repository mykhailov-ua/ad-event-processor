package affiliatestatus

import "testing"

func TestMapFromSchemaBody_mapsSale(t *testing.T) {
	goal, ok := MapFromSchemaBody([]byte(`{
		"version": 1,
		"status_map": {"sale": "approved"}
	}`), "sale")
	if !ok || goal != "approved" {
		t.Fatalf("goal=%q ok=%v", goal, ok)
	}
}

func TestMapFromSchemaBody_unknownStatus(t *testing.T) {
	_, ok := MapFromSchemaBody([]byte(`{"status_map":{"sale":"approved"}}`), "missing")
	if ok {
		t.Fatal("expected miss")
	}
}
