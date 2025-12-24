package loadbalancer

import "testing"

func TestRoundRobinCyclesServers(t *testing.T) {
	rr := NewRoundRobin([]string{"s1", "s2", "s3"})

	got := []string{}
	for i := 0; i < 5; i++ {
		s, err := rr.NextServer()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, s)
	}

	want := []string{"s1", "s2", "s3", "s1", "s2"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %s want %s", i, got[i], want[i])
		}
	}
}
