package registry

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

func TestAValueLingersAfterItIsDoneThenIsForgotten(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := New[int](time.Minute)
		done := make(chan struct{})
		if err := r.Add("a", 1, done); err != nil {
			t.Fatalf("an open registry refused a value: %v", err)
		}
		close(done)
		synctest.Sleep(59 * time.Second)
		if _, ok := r.Find("a"); !ok {
			t.Fatal("a done value was forgotten before it lingered")
		}
		synctest.Sleep(2 * time.Second)
		if _, ok := r.Find("a"); ok {
			t.Fatal("a done value was kept past its linger")
		}
	})
}

func TestADrainingRegistryRefusesValuesAndWaitsForThoseRunning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := New[int](time.Minute)
		done := make(chan struct{})
		r.Add("a", 1, done)
		drained := make(chan struct{})
		go func() {
			r.Drain(t.Context())
			close(drained)
		}()
		synctest.Wait()
		if err := r.Add("b", 2, make(chan struct{})); !errors.Is(err, ErrDraining) {
			t.Fatalf("a draining registry took a value: %v", err)
		}
		select {
		case <-drained:
			t.Fatal("the drain returned while a value still ran")
		default:
		}
		close(done)
		synctest.Wait()
		select {
		case <-drained:
		default:
			t.Fatal("the drain did not return once every value was done")
		}
	})
}

func TestAnIDInUseIsRefusedAndTheFirstValueKeepsIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := New[int](time.Minute)
		first := make(chan struct{})
		if err := r.Add("a", 1, first); err != nil {
			t.Fatal(err)
		}
		if err := r.Add("a", 2, make(chan struct{})); !errors.Is(err, ErrTaken) {
			t.Fatalf("a second value under a taken id: %v, want it refused", err)
		}
		close(first)
		synctest.Sleep(2 * time.Minute)
		if _, ok := r.Find("a"); ok {
			t.Fatal("the id was never released")
		}
		again := make(chan struct{})
		if err := r.Add("a", 3, again); err != nil {
			t.Fatalf("an id released after its linger: %v", err)
		}
		close(again)
	})
}
