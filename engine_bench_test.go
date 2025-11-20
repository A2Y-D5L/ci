package ci_test

import (
	"context"
	"testing"

	"github.com/a2y-d5l/ci"
	"github.com/a2y-d5l/ci/target"
)

// BenchmarkSimpleDAG benchmarks a simple linear dependency chain: A -> C
func BenchmarkSimpleDAG(b *testing.B) {
	a := target.New("A", "Base", func(ctx context.Context) error { return nil })
	c := target.New("C", "Final", func(ctx context.Context) error { return nil }, a)

	for i := 0; i < b.N; i++ {
		_, err := ci.RunTargets(context.Background(), c)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDiamondDAG benchmarks a diamond dependency graph:
//
//	    A
//	   / \
//	  B   C
//	   \ /
//	    D
func BenchmarkDiamondDAG(b *testing.B) {
	a := target.New("A", "Base", func(ctx context.Context) error { return nil })
	b1 := target.New("B", "Left", func(ctx context.Context) error { return nil }, a)
	c1 := target.New("C", "Right", func(ctx context.Context) error { return nil }, a)
	d := target.New("D", "Final", func(ctx context.Context) error { return nil }, b1, c1)

	for i := 0; i < b.N; i++ {
		_, err := ci.RunTargets(context.Background(), d)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWideDAG benchmarks a wide dependency graph with many parallel targets
func BenchmarkWideDAG(b *testing.B) {
	a := target.New("A", "Base", func(ctx context.Context) error { return nil })

	// Create 10 targets that all depend on A
	deps := make([]target.T, 10)
	for i := 0; i < 10; i++ {
		deps[i] = target.New(string(rune('B'+i)), "Parallel", func(ctx context.Context) error { return nil }, a)
	}

	// Create a final target that depends on all parallel targets
	final := target.New("Final", "Convergence", func(ctx context.Context) error { return nil }, deps...)

	for i := 0; i < b.N; i++ {
		_, err := ci.RunTargets(context.Background(), final)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDeepDAG benchmarks a deep linear dependency chain
func BenchmarkDeepDAG(b *testing.B) {
	// Create a chain of 20 targets: A -> B -> C -> ... -> T
	current := target.New("A", "Base", func(ctx context.Context) error { return nil })
	for i := 1; i < 20; i++ {
		current = target.New(string(rune('A'+i)), "Step", func(ctx context.Context) error { return nil }, current)
	}

	for i := 0; i < b.N; i++ {
		_, err := ci.RunTargets(context.Background(), current)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComplexDAG benchmarks a complex multi-level dependency graph
func BenchmarkComplexDAG(b *testing.B) {
	// Graph: F -> (D, E); D -> (B, C); E -> C; B -> A; C -> A
	a := target.New("A", "Base", func(ctx context.Context) error { return nil })
	b1 := target.New("B", "Level 2", func(ctx context.Context) error { return nil }, a)
	c := target.New("C", "Level 2", func(ctx context.Context) error { return nil }, a)
	d := target.New("D", "Level 3", func(ctx context.Context) error { return nil }, b1, c)
	e := target.New("E", "Level 3", func(ctx context.Context) error { return nil }, c)
	f := target.New("F", "Final", func(ctx context.Context) error { return nil }, d, e)

	for i := 0; i < b.N; i++ {
		_, err := ci.RunTargets(context.Background(), f)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIndependentTargets benchmarks multiple independent targets with no dependencies
func BenchmarkIndependentTargets(b *testing.B) {
	a := target.New("A", "Independent 1", func(ctx context.Context) error { return nil })
	b1 := target.New("B", "Independent 2", func(ctx context.Context) error { return nil })
	c := target.New("C", "Independent 3", func(ctx context.Context) error { return nil })
	d := target.New("D", "Independent 4", func(ctx context.Context) error { return nil })
	e := target.New("E", "Independent 5", func(ctx context.Context) error { return nil })

	for i := 0; i < b.N; i++ {
		_, err := ci.RunTargets(context.Background(), a, b1, c, d, e)
		if err != nil {
			b.Fatal(err)
		}
	}
}
