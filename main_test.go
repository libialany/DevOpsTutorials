package main

import "testing"

func TestCalculate(t *testing.T) {
	tests := []struct {
		name    string
		a, b    float64
		op      string
		want    float64
		wantErr bool
	}{
		{"add", 3, 4, "+", 7, false},
		{"subtract", 10, 4, "-", 6, false},
		{"multiply", 3, 5, "*", 15, false},
		{"divide", 10, 4, "/", 2.5, false},
		{"divide by zero", 1, 0, "/", 0, true},
		{"unknown operator", 1, 2, "%", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculate(tt.a, tt.op, tt.b)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}