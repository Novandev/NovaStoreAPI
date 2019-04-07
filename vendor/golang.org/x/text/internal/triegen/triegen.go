// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package triegen implements a code generator for a trie for associating
// unsigned integer values with UTF-8 encoded runes.
//
// Many of the go.text packages use tries for storing per-rune information.  A
// trie is especially useful if many of the runes have the same value. If this
// is the case, many blocks can be expected to be shared allowing for
// information on many runes to be stored in little space.
//
// As most of the lookups are done directly on []byte slices, the tries use the
// UTF-8 bytes directly for the lookup. This saves a conversion from UTF-8 to
// runes and contributes a little bit to better performance. It also naturally
// provides a fast path for ASCII.
//
// Space is also an issue. There are many code points defined in Unicode and as
// a result tables can get quite large. So every byte counts. The triegen
// package automatically chooses the smallest integer values to represent the
// tables. Compacters allow further compression of the trie by allowing for
// alternative representations of individual trie blocks.
//
// triegen allows generating multiple tries as a single structure. This is
// useful when, for example, one wants to generate tries for several languages
// that have a lot of values in common. Some existing libraries for
// internationalization store all per-language data as a dynamically loadable
// chunk. The go.text packages are designed with the assumption that the user
// typically wants to compile in support for all supported languages, in line
// with the approach common to Go to create a single standalone binary. The
// multi-root trie approach can give significant storage savings in this
// scenario.
//
// triegen generates both tables and code. The code is optimized to use the
// automatically chosen data types. The following code is generated for a Trie
// or multiple Tries named "foo":
//	- type fooTrie
//		The trie type.
//
//	- func newFooTrie(x int) *fooTrie
//		Trie constructor, where x is the index of the trie passed to Gen.
//
//	- func (t *fooTrie) lookup(s []byte) (v uintX, sz int)
//		The lookup method, where uintX is automatically chosen.
//
//	- func lookupString, lookupUnsafe and lookupStringUnsafe
//		Variants of the above.
//
//	- var fooValues and fooIndex and any tables generated by Compacters.
//		The core trie data.
//
//	- var fooTrieHandles
//		Indexes of starter blocks in case of multiple trie roots.
//
// It is recommended that users test the generated trie by checking the returned
// value for every rune. Such exhaustive tests are possible as the number of
// runes in Unicode is limited.
package triegen

// TODO: Arguably, the internally optimized data types would not have to be
// exposed in the generated API. We could also investigate not generating the
// code, but using it through a package. We would have to investigate the impact
// on performance of making such change, though. For packages like unicode/norm,
// small changes like this could tank performance.

import (
	"encoding/binary"
	"fmt"
	"hash/crc64"
	"io"
	"log"
	"unicode/utf8"
)

// builder builds a set of tries for associating values with runes. The set of
// tries can share common index and value blocks.
type builder struct {
	Name string

	// ValueType is the type of the trie values looked up.
	ValueType string

	// ValueSize is the byte size of the ValueType.
	ValueSize int

	// IndexType is the type of trie index values used for all UTF-8 bytes of
	// a rune except the last one.
	IndexType string

	// IndexSize is the byte size of the IndexType.
	IndexSize int

	// SourceType is used when generating the lookup functions. If the user
	// requests StringSupport, all lookup functions will be generated for
	// string input as well.
	SourceType string

	Trie []*Trie

	IndexBlocks []*node
	ValueBlocks [][]uint64
	Compactions []compaction
	Checksum    uint64

	ASCIIBlock   string
	StarterBlock string

	indexBlockIdx map[uint64]int
	valueBlockIdx map[uint64]nodeIndex
	asciiBlockIdx map[uint64]int

	// Stats are used to fill out the template.
	Stats struct {
		NValueEntries int
		NValueBytes   int
		NIndexEntries int
		NIndexBytes   int
		NHandleBytes  int
	}

	err error
}

// A nodeIndex encodes the index of a node, which is defined by the compaction
// which stores it and an index within the compaction. For internal nodes, the
// compaction is always 0.
type nodeIndex struct {
	compaction int
	index      int
}

// compaction keeps track of stats used for the compaction.
type compaction struct {
	c         Compacter
	blocks    []*node
	maxHandle uint32
	totalSize int

	// Used by template-based generator and thus exported.
	Cutoff  uint32
	Offset  uint32
	Handler string
}

func (b *builder) setError(err error) {
	if b.err == nil {
		b.err = err
	}
}

// An Option can be passed to Gen.
type Option func(b *builder) error

// Compact configures the trie generator to use the given Compacter.
func Compact(c Compacter) Option {
	return func(b *builder) error {
		b.Compactions = append(b.Compactions, compaction{
			c:       c,
			Handler: c.Handler() + "(n, b)"})
		return nil
	}
}

// Gen writes Go code for a shared trie lookup structure to w for the given
// Tries. The generated trie type will be called nameTrie. newNameTrie(x) will
// return the *nameTrie for tries[x]. A value can be looked up by using one of
// the various lookup methods defined on nameTrie. It returns the table size of
// the generated trie.
func Gen(w io.Writer, name string, tries []*Trie, opts ...Option) (sz int, err error) {
	// The index contains two dummy blocks, followed by the zero block. The zero
	// block is at offset 0x80, so that the offset for the zero block for
	// continuation bytes is 0.
	b := &builder{
		Name:        name,
		Trie:        tries,
		IndexBlocks: []*node{{}, {}, {}},
		Compactions: []compaction{{
			Handler: name + "Values[n<<6+uint32(b)]",
		}},
		// The 0 key in indexBlockIdx and valueBlockIdx is the hash of the zero
		// block.
		indexBlockIdx: map[uint64]int{0: 0},
		valueBlockIdx: map[uint64]nodeIndex{0: {}},
		asciiBlockIdx: map[uint64]int{},
	}
	b.Compactions[0].c = (*simpleCompacter)(b)

	for _, f := range opts {
		if err := f(b); err != nil {
			return 0, err
		}
	}
	b.build()
	if b.err != nil {
		return 0, b.err
	}
	if err = b.print(w); err != nil {
		return 0, err
	}
	return b.Size(), nil
}

// A Trie represents a single root node of a trie. A builder may build several
// overlapping tries at once.
type Trie struct {
	root *node

	hiddenTrie
}

// hiddenTrie contains values we want to be visible to the template generator,
// but hidden from the API documentation.
type hiddenTrie struct {
	Name         string
	Checksum     uint64
	ASCIIIndex   int
	StarterIndex int
}

// NewTrie returns a new trie root.
func NewTrie(name string) *Trie {
	return &Trie{
		&node{
			children: make([]*node, blockSize),
			values:   make([]uint64, utf8.RuneSelf),
		},
		hiddenTrie{Name: name},
	}
}

// Gen is a convenience wrapper around the Gen func passing t as the only trie
// and uses the name passed to NewTrie. It returns the size of the generated
// tables.
func (t *Trie) Gen(w io.Writer, opts ...Option) (sz int, err error) {
	return Gen(w, t.Name, []*Trie{t}, opts...)
}

// node is a node of the intermediate trie structure.
type node struct {
	// children holds this node's children. It is always of length 64.
	// A child node may be nil.
	children []*node

	// values contains the values of this node. If it is non-nil, this node is
	// either a root or leaf node:
	// For root nodes, len(values) == 128 and it maps the bytes in [0x00, 0x7F].
	// For leaf nodes, len(values) ==  64 and it maps the bytes in [0x80, 0xBF].
	values []uint64

	index nodeIndex
}

// Insert associates value with the given rune. Insert will panic if a non-zero
// value is passed for an invalid rune.
func (t *Trie) Insert(r rune, value uint64) {
	if value == 0 {
		return
	}
	s := string(r)
	if []rune(s)[0] != r && value != 0 {
		// Note: The UCD tables will always assign what amounts to a zero value
		// to a surrogate. Allowing a zero value for an illegal rune allows
		// users to iterate over [0..MaxRune] without having to explicitly
		// exclude surrogates, which would be tedious.
		panic(fmt.Sprintf("triegen: non-zero value for invalid rune %U", r))
	}
	if len(s) == 1 {
		// It is a root node value (ASCII).
		t.root.values[s[0]] = value
		return
	}

	n := t.root
	for ; len(s) > 1; s = s[1:] {
		if n.children == nil {
			n.children = make([]*node, blockSize)
		}
		p := s[0] % blockSize
		c := n.children[p]
		if c == nil {
			c = &node{}
			n.children[p] = c
		}
		if len(s) > 2 && c.values != nil {
			log.Fatalf("triegen: insert(%U): found internal node with values", r)
		}
		n = c
	}
	if n.values == nil {
		n.values = make([]uint64, blockSize)
	}
	if n.children != nil {
		log.Fatalf("triegen: insert(%U): found leaf node that also has child nodes", r)
	}
	n.values[s[0]-0x80] = value
}

// Size returns the number of bytes the generated trie will take to store. It
// needs to be exported as it is used in the templates.
func (b *builder) Size() int {
	// Index blocks.
	sz := len(b.IndexBlocks) * blockSize * b.IndexSize

	// Skip the first compaction, which represents the normal value blocks, as
	// its totalSize does not account for the ASCII blocks, which are managed
	// separately.
	sz += len(b.ValueBlocks) * blockSize * b.ValueSize
	for _, c := range b.Compactions[1:] {
		sz += c.totalSize
	}

	// TODO: this computation does not account for the fixed overhead of a using
	// a compaction, either code or data. As for data, though, the typical
	// overhead of data is in the order of bytes (2 bytes for cases). Further,
	// the savings of using a compaction should anyway be substantial for it to
	// be worth it.

	// For multi-root tries, we also need to account for the handles.
	if len(b.Trie) > 1 {
		sz += 2 * b.IndexSize * len(b.Trie)
	}
	return sz
}

func (b *builder) build() {
	// Compute the sizes of the values.
	var vmax uint64
	for _, t := range b.Trie {
		vmax = maxValue(t.root, vmax)
	}
	b.ValueType, b.ValueSize = getIntType(vmax)

	// Compute all block allocations.
	// TODO: first compute the ASCII blocks for all tries and then the other
	// nodes. ASCII blocks are more restricted in placement, as they require two
	// blocks to be placed consecutively. Processing them first may improve
	// sharing (at least one zero block can be expected to be saved.)
	for _, t := range b.Trie {
		b.Checksum += b.buildTrie(t)
	}

	// Compute the offsets for all the Compacters.
	offset := uint32(0)
	for i := range b.Compactions {
		c := &b.Compactions[i]
		c.Offset = offset
		offset += c.maxHandle + 1
		c.Cutoff = offset
	}

	// Compute the sizes of indexes.
	// TODO: different byte positions could have different sizes. So far we have
	// not found a case where this is beneficial.
	imax := uint64(b.Compactions[len(b.Compactions)-1].Cutoff)
	for _, ib := range b.IndexBlocks {
		if x := uint64(ib.index.index); x > imax {
			imax = x
		}
	}
	b.IndexType, b.IndexSize = getIntType(imax)
}

func maxValue(n *node, max uint64) uint64 {
	if n == nil {
		return max
	}
	for _, c := range n.children {
		max = maxValue(c, max)
	}
	for _, v := range n.values {
		if max < v {
			max = v
		}
	}
	return max
}

func getIntType(v uint64) (string, int) {
	switch {
	case v < 1<<8:
		return "uint8", 1
	case v < 1<<16:
		return "uint16", 2
	case v < 1<<32:
		return "uint32", 4
	}
	return "uint64", 8
}

const (
	blockSize = 64

	// Subtract two blocks to offset 0x80, the first continuation byte.
	blockOffset = 2

	// Subtract three blocks to offset 0xC0, the first non-ASCII starter.
	rootBlockOffset = 3
)

var crcTable = crc64.MakeTable(crc64.ISO)

func (b *builder) buildTrie(t *Trie) uint64 {
	n := t.root

	// Get the ASCII offset. For the first trie, the ASCII block will be at
	// position 0.
	hasher := crc64.New(crcTable)
	binary.Write(hasher, binary.BigEndian, n.values)
	hash := hasher.Sum64()

	v, ok := b.asciiBlockIdx[hash]
	if !ok {
		v = len(b.ValueBlocks)
		b.asciiBlockIdx[hash] = v

		b.ValueBlocks = append(b.ValueBlocks, n.values[:blockSize], n.values[blockSize:])
		if v == 0 {
			// Add the zero block at position 2 so that it will be assigned a
			// zero reference in the lookup blocks.
			// TODO: always do this? This would allow us to remove a check from
			// the trie lookup, but at the expense of extra space. Analyze
			// performance for unicode/norm.
			b.ValueBlocks = append(b.ValueBlocks, make([]uint64, blockSize))
		}
	}
	t.ASCIIIndex = v

	// Compute remaining offsets.
	t.Checksum = b.computeOffsets(n, true)
	// We already subtracted the normal blockOffset from the index. Subtract the
	// difference for starter bytes.
	t.StarterIndex = n.index.index - (rootBlockOffset - blockOffset)
	return t.Checksum
}

func (b *builder) computeOffsets(n *node, root bool) uint64 {
	// For the first trie, the root lookup block will be at position 3, which is
	// the offset for UTF-8 non-ASCII starter bytes.
	first := len(b.IndexBlocks) == rootBlockOffset
	if first {
		b.IndexBlocks = append(b.IndexBlocks, n)
	}

	// We special-case the cases where all values recursively are 0. This allows
	// for the use of a zero block to which all such values can be directed.
	hash := uint64(0)
	if n.children != nil || n.values != nil {
		hasher := crc64.New(crcTable)
		for _, c := range n.children {
			var v uint64
			if c != nil {
				v = b.computeOffsets(c, false)
			}
			binary.Write(hasher, binary.BigEndian, v)
		}
		binary.Write(hasher, binary.BigEndian, n.values)
		hash = hasher.Sum64()
	}

	if first {
		b.indexBlockIdx[hash] = rootBlockOffset - blockOffset
	}

	// Compacters don't apply to internal nodes.
	if n.children != nil {
		v, ok := b.indexBlockIdx[hash]
		if !ok {
			v = len(b.IndexBlocks) - blockOffset
			b.IndexBlocks = append(b.IndexBlocks, n)
			b.indexBlockIdx[hash] = v
		}
		n.index = nodeIndex{0, v}
	} else {
		h, ok := b.valueBlockIdx[hash]
		if !ok {
			bestI, bestSize := 0, blockSize*b.ValueSize
			for i, c := range b.Compactions[1:] {
				if sz, ok := c.c.Size(n.values); ok && bestSize > sz {
					bestI, bestSize = i+1, sz
				}
			}
			c := &b.Compactions[bestI]
			c.totalSize += bestSize
			v := c.c.Store(n.values)
			if c.maxHandle < v {
				c.maxHandle = v
			}
			h = nodeIndex{bestI, int(v)}
			b.valueBlockIdx[hash] = h
		}
		n.index = h
	}
	return hash
}
