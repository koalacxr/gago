package gago

import (
	"math/rand"
	"testing"
	"time"
)

func TestCastFloat(t *testing.T) {
	var (
		nbGenes   = 4
		source    = rand.NewSource(time.Now().UnixNano())
		generator = rand.New(source)
		indi      = makeIndividual(nbGenes)
		init      = InitUniformF{-5.0, 5.0}
	)
	init.apply(&indi, generator)
	// Check the casting doesn't change the length of the genome
	var casted = indi.Genome.CastFloat()
	if len(casted) != nbGenes {
		t.Error("Casting changed the genome's length")
	}
}

func TestCastString(t *testing.T) {
	var (
		nbGenes   = 4
		source    = rand.NewSource(time.Now().UnixNano())
		generator = rand.New(source)
		indi      = makeIndividual(nbGenes)
		init      = InitUniformS{[]string{"T", "E", "S", "T"}}
	)
	init.apply(&indi, generator)
	// Check the casting doesn't change the length of the genome
	var casted = indi.Genome.CastString()
	if len(casted) != nbGenes {
		t.Error("Casting changed the genome's length")
	}
}
