package main

import (
	"math"
	"os"
	"testing"
)

func TestReadHeaders(t *testing.T) {
	content := "Time,SignalA,SignalB\n1,2,3\n4,5,6"
	tmpfile, err := os.CreateTemp("", "test*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	headers, err := readCSVHeaders(tmpfile.Name())
	if err != nil {
		t.Errorf("readCSVHeaders failed: %v", err)
	}

	expected := []string{"Time", "SignalA", "SignalB"}
	if len(headers) != len(expected) {
		t.Fatalf("expected %d headers, got %d", len(expected), len(headers))
	}

	for i, h := range headers {
		if h != expected[i] {
			t.Errorf("expected header %d to be %s, got %s", i, expected[i], h)
		}
	}
}

func TestLoadCSVData(t *testing.T) {
	content := "Time,SignalA\n1.0,2.0\n3.0,invalid\n5.0,6.0"
	tmpfile, err := os.CreateTemp("", "testdata*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	headers := []string{"Time", "SignalA"}
	data, err := loadCSVData(tmpfile.Name(), headers)
	if err != nil {
		t.Fatalf("loadCSVData failed: %v", err)
	}

	if len(data["Time"]) != 3 {
		t.Errorf("expected 3 rows in Time, got %d", len(data["Time"]))
	}

	if data["Time"][0] != 1.0 || data["Time"][1] != 3.0 || data["Time"][2] != 5.0 {
		t.Errorf("Time data mismatch: %v", data["Time"])
	}

	if data["SignalA"][0] != 2.0 || !math.IsNaN(data["SignalA"][1]) || data["SignalA"][2] != 6.0 {
		t.Errorf("SignalA data mismatch (NaN handling): %v", data["SignalA"])
	}
}

// End of tests
