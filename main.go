package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func ExpectOk(err error) {
	if err != nil {
		log.Fatalf("Unexpected fatal error: %s\n", err)
	}
}

var ColorList []string

const StartingColor = 10

func LoadColorList(filename string) error {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &ColorList)
	return err
}

type Graph struct {
	AdjecencyList [][]int
	Colors        []int
}

func NewRandomGraph(nodeCount int, prob float32) Graph {
	g := Graph{}

	g.AdjecencyList = make([][]int, nodeCount)
	g.Colors = make([]int, nodeCount)

	for i := 0; i < nodeCount; i++ {
		for j := i + 1; j < nodeCount; j++ {
			if rand.Float32() < prob {
				g.AdjecencyList[i] = append(g.AdjecencyList[i], j)
			}
		}
	}

	return g
}

func LoadGraph(filename string) (*Graph, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	g := Graph{}

	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		switch line[0] {
		case 'c':
			continue
		case 'p':
			tokens := strings.Split(line, " ")
			nodeCount, err := strconv.ParseInt(tokens[2], 10, 32)
			if err != nil {
				return nil, err
			}
			g.AdjecencyList = make([][]int, nodeCount)
			g.Colors = make([]int, nodeCount)
		case 'e':
			tokens := strings.Split(line, " ")
			first, err := strconv.ParseInt(tokens[1], 10, 32)
			if err != nil {
				return nil, err
			}
			second, err := strconv.ParseInt(tokens[2], 10, 32)
			if err != nil {
				return nil, err
			}
			g.AdjecencyList[first-1] = append(g.AdjecencyList[first-1], int(second-1))
		}
	}

	return &g, nil
}

func (g *Graph) Save(filename string) error {
	bytes, err := json.Marshal(g)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, bytes, 0600)
	return err
}

func (g *Graph) SaveGraphViz(filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil
	}
	defer file.Close()

	_, err = file.WriteString("graph {\n\tnode [colorscheme=accent8]\n")
	if err != nil {
		return err
	}

	nodeCount := g.NodeCount()
	for i := 0; i < nodeCount; i++ {
		for _, j := range g.AdjecencyList[i] {
			_, err = file.WriteString(fmt.Sprintf("\t%d -- %d\n", i, j))
			if err != nil {
				return err
			}
		}
	}

	for i := 0; i < nodeCount; i++ {
		_, err = file.WriteString(fmt.Sprintf(
			"\t%d [style=filled, color=%d]\n",
			i,
			g.Colors[i]+1,
		))
	}

	_, err = file.WriteString("}\n")
	return err
}

func (g *Graph) NodeCount() int {
	return len(g.AdjecencyList)
}

type Chromosome = []int
type Population = []Chromosome

type GraphColoringSolver struct {
	Graph     Graph
	NumColors int

	population Population
}

func NewGraphColoringSolver(graph Graph, numColors int) GraphColoringSolver {
	return GraphColoringSolver{
		Graph:     graph,
		NumColors: numColors,
	}
}

func (solver *GraphColoringSolver) RandomPopulation(size int) Population {
	pop := make(Population, size)

	nodeCount := solver.Graph.NodeCount()
	for i := 0; i < size; i++ {
		chr := make(Chromosome, nodeCount)
		for j := 0; j < nodeCount; j++ {
			chr[j] = rand.Intn(solver.NumColors)
		}
		pop[i] = chr
	}

	return pop
}

type GraphColoringSolution struct {
	Coloring Chromosome
	Score    int
}

func (solution *GraphColoringSolution) Save(filename string) error {
	bytes, err := json.Marshal(solution)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, bytes, 0600)
}

func (solver *GraphColoringSolver) SelectParents(population Population) []Chromosome {
	popSize := len(population)

	const parentsCount int = 2
	var parents []Chromosome
	usedParents := make(map[int]struct{})

	for i := 0; i < parentsCount; i++ {
		var parentIndex int
		for j := 0; j < 10; j++ {
			parentIndex = rand.Intn(popSize)
			_, exists := usedParents[parentIndex]
			if !exists {
				usedParents[parentIndex] = struct{}{}
				break
			}
		}
		parents = append(parents, population[parentIndex])
	}

	return parents
}

func (solver *GraphColoringSolver) Crossover(parents []Chromosome) Chromosome {
	var res Chromosome
	chromosomeLength := len(parents[0])
	partsCount := len(parents)
	partLength := chromosomeLength / partsCount

	for currentIndex := 0; currentIndex < chromosomeLength; {
		nextIndex := currentIndex + partLength
		if chromosomeLength < nextIndex {
			nextIndex = chromosomeLength
		}
		parentIndex := rand.Intn(len(parents))
		for i := currentIndex; i < nextIndex; i++ {
			res = append(res, parents[parentIndex][i])
		}

		currentIndex = nextIndex
	}

	return res
}

func (solver *GraphColoringSolver) Mutate(child Chromosome) Chromosome {
	mutationProb := 1.0 / float32(len(child))

	for i := 0; i < len(child); i++ {
		if rand.Float32() < mutationProb {
			child[i] = rand.Intn(solver.NumColors)
		}
	}

	return child
}

func (solver *GraphColoringSolver) CalculateFitness(chromosome Chromosome) int {
	score := 0
	for i := 0; i < solver.Graph.NodeCount(); i++ {
		for _, j := range solver.Graph.AdjecencyList[i] {
			if chromosome[i] == chromosome[j] {
				score += 1
			}
		}
	}
	return score / 2
}

type scoredChromosome struct {
	chromosome Chromosome
	score      int
}

func (solver *GraphColoringSolver) Solve(numIterations int, popSize int) GraphColoringSolution {
	population := solver.RandomPopulation(popSize)

	childrenPopSize := 2 * popSize

	for iteration := 0; iteration < numIterations; iteration++ {
		var scoredPopulation []scoredChromosome
		for childIndex := 0; childIndex < childrenPopSize; childIndex++ {
			parents := solver.SelectParents(population)
			child := solver.Crossover(parents)
			mutatedChild := solver.Mutate(child)
			score := solver.CalculateFitness(mutatedChild)
			scoredPopulation = append(scoredPopulation, scoredChromosome{
				chromosome: mutatedChild,
				score:      score,
			})
		}
		sort.Slice(scoredPopulation, func(i int, j int) bool {
			return scoredPopulation[i].score < scoredPopulation[j].score
		})
		for i := 0; i < popSize; i++ {
			population[i] = scoredPopulation[i].chromosome
		}
		bestScore := scoredPopulation[0].score

		if iteration%100 == 0 {
			log.Printf("Iteration %d: Score %d\n", iteration, bestScore)
		}
		if bestScore == 0 {
			break
		}
	}

	return GraphColoringSolution{
		Coloring: population[0],
		Score:    solver.CalculateFitness(population[0]),
	}
}

func main() {
	rand.Seed(time.Now().UnixMicro())

	// ExpectOk(LoadColorList("colors.json"))

	// n := 1000
	// g := NewRandomGraph(n, 3.0/float32(n))
	// ExpectOk(g.Save("graph.json"))
	// ExpectOk(g.SaveGraphViz("graph-viz.dot"))

	g, err := LoadGraph("dataset/data/queen7_7.col")
	ExpectOk(err)

	solver := NewGraphColoringSolver(*g, 7)
	solution := solver.Solve(100000, 200)

	outputFilename := "result.json"
	ExpectOk(solution.Save(outputFilename))
	g.Colors = solution.Coloring
	ExpectOk(g.SaveGraphViz("solution-viz.dot"))

	log.Printf("Best coloring score: %d. Coloring saved in file %s\n", solution.Score, outputFilename)
}
