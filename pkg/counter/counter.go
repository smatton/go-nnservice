package counter

import (
	"sort"
)

type Counts struct {
	Key   []byte
	Count int
	Dist  float32
}

type Counter struct {
	dict       map[string]Counts
	CountSlice []Counts
}

func NewCounter() *Counter {
	var c Counter
	c.dict = make(map[string]Counts)
	return &c
}
func (c *Counter) Add(key []byte, distance float32) {
	count, ok := c.dict[string(key)]
	if ok {
		c.dict[string(key)] = Counts{Key: count.Key, Count: count.Count + 1, Dist: count.Dist}
	} else {
		c.dict[string(key)] = Counts{Key: key, Count: 1, Dist: distance}
	}
}

func (c *Counter) Sort() {
	countSlice := make([]Counts, len(c.dict))
	i := 0
	for _, v := range c.dict {
		countSlice[i] = v
		i++
	}

	c.CountSlice = countSlice
	sort.Sort(ByCount(c.CountSlice))
}

func (c *Counter) Most() Counts {
	c.Sort()
	return c.CountSlice[0]
}

type ByCount []Counts

func (a ByCount) Len() int      { return len(a) }
func (a ByCount) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// we want the highest count first
func (a ByCount) Less(i, j int) bool { return a[i].Count > a[j].Count }
