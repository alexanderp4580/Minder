package domain

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOldStoredCaptureWithStringQuestionsLoads(t *testing.T) {
	var c Capture
	if err := json.Unmarshal([]byte(`{"id":"x","questions":["What date?",{"id":"q2","text":"Where?","options":["Skip","Home"]}]}`), &c); err != nil {
		t.Fatal(err)
	}
	want := []CaptureQuestion{{Text: "What date?"}, {ID: "q2", Text: "Where?", Options: []string{"Skip", "Home"}}}
	if !reflect.DeepEqual(c.Questions, want) {
		t.Fatal(c.Questions)
	}
	out, _ := json.Marshal(c.Questions[0])
	if string(out) != `{"text":"What date?"}` {
		t.Fatal(string(out))
	}
}
