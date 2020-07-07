//neighbors golang hnswlib implementation here https://github.com/Bithack/go-hnsw
//to support string labels as suggested in https://github.com/nmslib/hnswlib/tree/master/examples/pyw_hnswlib.py
package neighbors

import (
	"compress/gzip"
	"encoding/gob"
	"math"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"

	hnsw "github.com/Bithack/go-hnsw"
)

type Index struct {
	sync.RWMutex
	Hnsw         *hnsw.Hnsw
	currentIndex uint32
	labelDict    map[uint32][]byte
	efSearch     int
	Normalized   bool
}

func NewIndex(M, efConstruction, max_elements, dim int, normalize bool) *Index {
	var Ind Index

	zero := randomPoint(dim)

	Ind.Hnsw = hnsw.New(M, efConstruction, zero)
	Ind.Hnsw.Grow(max_elements)
	Ind.labelDict = make(map[uint32][]byte)
	Ind.Normalized = normalize

	return &Ind

}

func (index *Index) Save(filename string) error {
	index.Lock()
	defer index.Unlock()
	err := index.Hnsw.Save(filename)
	if err != nil {
		return err
	}

	labeldictname := filename + ".labs.gz"
	err = index.savelabelDict(labeldictname)
	if err != nil {
		return err
	}
	return nil
}

func (index *Index) Load(filename string) error {
	newindex, _, err := hnsw.Load(filename)
	index.Hnsw = newindex
	if err != nil {
		return err
	}

	labeldictname := filename + ".labs.gz"
	err = index.loadLabelDict(labeldictname)
	if err != nil {
		return err
	}
	return nil
}

//Add overload the add operation to support string labels
func (index *Index) Insert(point []float32, label []byte) {
	var pt hnsw.Point

	if index.Normalized {
		pt = Normalize(point)
	} else {
		pt = point
	}

	atomic.AddUint32(&index.currentIndex, 1)

	index.Hnsw.Add(pt, index.currentIndex)

	index.labelDict[index.currentIndex] = label

}

func (index *Index) SetEf(efSearch int) {
	index.efSearch = efSearch
}

func (index *Index) Grow(size int) {
	index.Hnsw.Grow(size)
}

func (index *Index) Search(point []float32, K int) ([][]byte, []float32) {
	var pt hnsw.Point
	if index.Normalized {
		pt = Normalize(point)
	} else {
		pt = point
	}

	label := make([][]byte, K)
	distances := make([]float32, K)

	result := index.Hnsw.Search(pt, index.efSearch, K)

	j := 0
	for result.Len() > 0 {
		i := result.Pop()
		label[j], _ = index.labelDict[i.ID]
		distances[j] = i.D
		j++
	}
	return label, distances
}

func (index *Index) savelabelDict(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	fz := gzip.NewWriter(f)
	defer fz.Close()

	labels := LabelDict{LabelDict: index.labelDict, CurrentIndex: index.currentIndex, Normalized: index.Normalized}
	e := gob.NewEncoder(fz)
	err = e.Encode(labels)
	if err != nil {
		return err
	}

	return nil
}

func (index *Index) loadLabelDict(filename string) error {

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	fz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	labels := LabelDict{}
	d := gob.NewDecoder(fz)
	err = d.Decode(&labels)
	if err != nil {
		return err
	}

	index.labelDict = labels.LabelDict
	index.currentIndex = labels.CurrentIndex
	index.Normalized = labels.Normalized

	return nil
}

//MajorityVote returns the most often found label and if there are ties
// then return the label with least distance
// func MajorityVote(labels [][]byte, distances []float32) ([]byte, int, float32) {
// 	cnter := counter.NewCounter()
// 	for i, label := range labels {
// 		cnter.Add(label, distances[i])
// 	}
// 	most := cnter.Most()
// 	// if ties get closest
// 	if most.Count == 1 {
// 		return labels[0], 1, distances[0]
// 	}
//
// 	// check for other ties
// 	closest := most
// 	for j, cnt := range cnter.CountSlice {
// 		if most.Count >= cnt.Count {
// 			if cnt.Dist < most.Dist {
// 				closest = cnt
// 			}
// 		} else {
// 			return closest.Key, closest.Count, closest.Dist
// 		}
//
// 	}
//
// 	return
// }

func randomPoint(dim int) hnsw.Point {
	var v hnsw.Point = make([]float32, dim)
	for i := range v {
		v[i] = rand.Float32()
	}
	return v
}

type LabelDict struct {
	CurrentIndex uint32
	Normalized   bool
	LabelDict    map[uint32][]byte
}

//Normalized return unit vector for the float slice passed
//by using unit vector points the euclidean distance will be proportional to
// the cosine distance of the two vectors
func Normalize(pt []float32) []float32 {
	normalized := make([]float32, len(pt))

	mag := float32(Magnitude(pt))

	for i, v := range pt {
		normalized[i] = v / mag
	}

	return normalized
}

func Magnitude(pt []float32) float64 {
	var sum float32
	for _, v := range pt {
		sum += v * v
	}

	return math.Sqrt(float64(sum))
}
