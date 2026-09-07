package expr

import "testing"

func TestBool(t *testing.T) {
	cases := []struct {
		name string
		expr string
		vars Vars
		want bool
	}{
		{
			name: "simple field comparison",
			expr: `record.status == "Draft"`,
			vars: Vars{Record: map[string]any{"status": "Draft"}},
			want: true,
		},
		{
			name: "old vs record comparison, changed",
			expr: `record.status != old.status`,
			vars: Vars{Record: map[string]any{"status": "Approved"}, Old: map[string]any{"status": "Draft"}},
			want: true,
		},
		{
			name: "old vs record comparison, unchanged",
			expr: `record.status != old.status`,
			vars: Vars{Record: map[string]any{"status": "Draft"}, Old: map[string]any{"status": "Draft"}},
			want: false,
		},
		{
			name: "current_user",
			expr: `current_user == record.owner`,
			vars: Vars{Record: map[string]any{"owner": "usr_1"}, CurrentUser: "usr_1"},
			want: true,
		},
		{
			name: "numeric comparison",
			expr: `double(record.amount) > 1000.0`,
			vars: Vars{Record: map[string]any{"amount": 1500.0}},
			want: true,
		},
		{
			name: "syntax error fails closed",
			expr: `record.status == `,
			vars: Vars{Record: map[string]any{"status": "Draft"}},
			want: false,
		},
		{
			name: "missing field on empty old fails closed",
			expr: `old.status == "Draft"`,
			vars: Vars{Record: map[string]any{"status": "Draft"}},
			want: false,
		},
		{
			name: "no I/O function available",
			expr: `http.get("https://example.com") != ""`,
			vars: Vars{},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Bool(tc.expr, tc.vars); got != tc.want {
				t.Errorf("Bool(%q) = %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestEvalValue(t *testing.T) {
	v, err := Eval(`double(record.price) * double(record.factor)`, Vars{Record: map[string]any{"price": 10.0, "factor": 3.0}})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	f, ok := v.(float64)
	if !ok || f != 30.0 {
		t.Errorf("Eval = %v (%T), want 30.0", v, v)
	}
}

func TestCompile(t *testing.T) {
	if err := Compile(`record.status == "Draft"`); err != nil {
		t.Errorf("Compile valid expression failed: %v", err)
	}
	if err := Compile(`record.status ==`); err == nil {
		t.Error("Compile invalid expression should have failed")
	}
}
