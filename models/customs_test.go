package models

import (
	"encoding/json"
	"testing"
)

func TestCustomsItemInput_OmitsEmptyExportClassification(t *testing.T) {
	for _, classification := range []string{"EAR99", "5A992", ""} {
		t.Run(classification, func(t *testing.T) {
			input := CustomsItemInput{Description: "Graded Cards", ECCNEAR99: classification}
			data, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			value, exists := fields["eccn_ear99"]
			if classification == "" {
				if exists {
					t.Error("empty export classification must be omitted")
				}
				return
			}
			if string(value) != `"`+classification+`"` {
				t.Errorf("eccn_ear99 = %s; want %q", value, classification)
			}
		})
	}
}

func TestCustomsItem_DecodesExportClassification(t *testing.T) {
	var item CustomsItem
	err := json.Unmarshal([]byte(`{"object_id":"item-1","description":"Graded Cards","eccn_ear99":"EAR99"}`), &item)
	if err != nil {
		t.Fatal(err)
	}
	if item.ECCNEAR99 != "EAR99" {
		t.Errorf("ECCNEAR99 = %q; want EAR99", item.ECCNEAR99)
	}
	if item.Description != "Graded Cards" || item.ObjectID != "item-1" {
		t.Errorf("unexpected customs item: %+v", item)
	}
}
