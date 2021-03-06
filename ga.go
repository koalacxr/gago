package gago

import (
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"
)

// A Topology holds all the information relative to the size of a GA.
type Topology struct {
	NPopulations int // Number of populations
	NSpecies     int // Number of species each population is split into
	NIndividuals int // Initial number of individuals in each population
}

// Validate the properties of a Topology.
func (topo Topology) Validate() error {
	if topo.NPopulations < 1 {
		return errors.New("'NPopulations' should be higher or equal to 1")
	}
	if topo.NSpecies < 0 {
		return errors.New("'NSpecies' should be higher or equal to 1 if provided")
	}
	if topo.NIndividuals < 1 {
		return errors.New("'NIndividuals' should be higher or equal to 1")
	}
	return nil
}

// A GA contains population which themselves contain individuals.
type GA struct {
	// Fields that are provided by the user
	MakeGenome   GenomeMaker
	Topology     Topology
	Model        Model
	Migrator     Migrator
	MigFrequency int // Frequency at which migrations occur
	Logger       *log.Logger

	// Fields that are generated at runtime
	Populations Populations
	Best        Individual // Overall best individual (dummy initialization at the beginning)
	Age         time.Duration
	Generations int
	rng         *rand.Rand
}

// Validate the parameters of a GA to ensure it will run correctly; some
// settings or combination of settings may be incoherent during runtime.
func (ga GA) Validate() error {
	// Check the GenomeMaker presence
	if ga.MakeGenome == nil {
		return errors.New("'GenomeMaker' cannot be nil")
	}
	// Check the topology is valid
	var topoErr = ga.Topology.Validate()
	if topoErr != nil {
		return topoErr
	}
	// Check the model presence
	if ga.Model == nil {
		return errors.New("'Model' cannot be nil")
	}
	// Check the model is valid
	var modelErr = ga.Model.Validate()
	if modelErr != nil {
		return modelErr
	}
	// Check the migration frequency in the presence of a migrator
	if ga.Migrator != nil && ga.MigFrequency < 1 {
		return errors.New("'MigFrequency' should be strictly higher than 0")
	}
	// No error
	return nil
}

// Find the best individual in each population and then compare the best overall
// individual to the current best individual. This method supposes that the
// populations have been preemptively sorted by fitness incresingly; this way
// checking the first individual of each population is sufficient.
func (ga *GA) findBest() {
	for _, pop := range ga.Populations {
		var best = pop.Individuals[0]
		if best.Fitness < ga.Best.Fitness {
			ga.Best = best
		}
	}
}

// Initialize each population in the GA and assign an initial fitness to each
// individual in each population. Running Initialize after running Enhance will
// reset the GA entirely.
func (ga *GA) Initialize() {
	ga.Populations = make([]Population, ga.Topology.NPopulations)
	ga.rng = makeRandomNumberGenerator()
	var wg sync.WaitGroup
	for i := range ga.Populations {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			// Generate a population
			ga.Populations[j] = makePopulation(ga.Topology.NIndividuals, ga.MakeGenome, j)
			// Evaluate it's individuals
			ga.Populations[j].Individuals.Evaluate()
			// Sort it's individuals
			ga.Populations[j].Individuals.Sort()
			// Log current statistics if a logger has been provided
			if ga.Logger != nil {
				go ga.Populations[j].Log(ga.Logger)
			}
		}(i)
	}
	wg.Wait()
	// The initial best individual is initialized randomly
	ga.Best = MakeIndividual(ga.MakeGenome(makeRandomNumberGenerator()))
	ga.findBest()
}

// Enhance each population in the GA. The population level operations are done
// in parallel with a wait group. After all the population operations have been
// run, the GA level operations are run.
func (ga *GA) Enhance() {
	var start = time.Now()
	ga.Generations++
	// Migrate the individuals between the populations if there are enough
	// populations, there is a migrator and the migration frequency divides the
	// generation count
	if ga.Topology.NPopulations > 1 && ga.Migrator != nil && ga.Generations%ga.MigFrequency == 0 {
		ga.Migrator.Apply(ga.Populations, ga.rng)
	}
	// Use a wait group to enhance the populations in parallel
	var wg sync.WaitGroup
	for i := range ga.Populations {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			// Apply speciation if a positive number of species has been speficied
			if ga.Topology.NSpecies > 0 {
				var species = ga.Populations[j].speciate(ga.Topology.NSpecies)
				// Apply the evolution model to each cluster
				for k := range species {
					ga.Model.Apply(&species[k])
				}
				// Merge each cluster back into the original population
				ga.Populations[j].Individuals = species.merge()
			} else {
				// Else apply the evolution model to the entire population
				ga.Model.Apply(&ga.Populations[j])
			}
			// Evaluate and sort
			ga.Populations[j].Individuals.Evaluate()
			ga.Populations[j].Individuals.Sort()
			ga.Populations[j].Age += time.Since(start)
			ga.Populations[j].Generations++
			// Log current statistics if a logger has been provided
			if ga.Logger != nil {
				go ga.Populations[j].Log(ga.Logger)
			}
		}(i)
	}
	wg.Wait()
	// Check if there is an individual that is better than the current one
	ga.findBest()
	ga.Age += time.Since(start)
}
