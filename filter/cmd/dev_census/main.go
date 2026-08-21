// Command census reconstructs the shape of the NORMALIZED feed from the data
// itself and prints it as an annotated tree.
//
// This is the JSON counterpart to the processor's census (which reads the raw
// FIXM XML). The processor emits one Kafka message per envelope, each a JSON
// array of flight objects, so this walks every element of that array as one
// flight and renders what the normalized JSON actually looks like: nesting,
// presence, cardinality, the JSON type of every field, and a real sample value
// for every leaf — so the output can be read straight down while writing the
// structs the redis / timescale / backend consumers will unmarshal into.
//
// Reads the partition directly rather than joining a consumer group, so it
// always starts from the beginning and never disturbs the filter's offsets.
// Safe to run repeatedly.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

// node is one field in the reconstructed tree. Unlike the XML census there are
// no attributes — a JSON value is an object, an array, or a scalar leaf — so a
// node carries the set of JSON kinds observed at its path plus, for leaves,
// sample and enumerated values.
type node struct {
	name string

	// flights this field appeared in at all
	flights int

	// largest cardinality seen in a single flight: for an array field this is
	// the array length, so >1 means it is a slice
	maxRepeat int

	// JSON kinds observed at this path, e.g. {"object":N} or {"array":N,"string":M}
	kinds map[string]int

	// leaf content, if this path ever held a scalar
	hasLeaf bool
	sample  string
	values  map[string]int

	children map[string]*node

	// insertion order, so siblings print in the order first seen rather than
	// alphabetically -- closer to how the document reads
	order []string
}

func newNode(name string) *node {
	return &node{
		name:     name,
		kinds:    map[string]int{},
		values:   map[string]int{},
		children: map[string]*node{},
	}
}

func (n *node) child(name string) *node {
	c, ok := n.children[name]
	if !ok {
		c = newNode(name)
		n.children[name] = c
		n.order = append(n.order, name)
	}
	return c
}

type census struct {
	envelopes int
	flights   int
	root      *node

	// per-flight presence + max cardinality, applied and cleared by flush
	seen map[*node]int
}

func newCensus() *census {
	return &census{
		root: newNode("flight"),
		seen: map[*node]int{},
	}
}

func (c *census) walk(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber() // keep numbers faithful (no float rounding in samples)

	var top interface{}
	if err := dec.Decode(&top); err != nil {
		return err
	}

	switch t := top.(type) {
	case []interface{}: // the normal case: an array of flights
		for _, f := range t {
			c.flights++
			c.observe(c.root, f)
			c.flush()
		}
	case map[string]interface{}: // tolerate a bare single flight
		c.flights++
		c.observe(c.root, t)
		c.flush()
	default:
		return fmt.Errorf("top-level JSON is %T, expected array or object", top)
	}
	return nil
}

// observe records that node n took on value v in the current flight, updating
// kinds, cardinality, samples, and recursing into structure.
func (c *census) observe(n *node, v interface{}) {
	switch val := v.(type) {

	case map[string]interface{}:
		c.mark(n, 1)
		n.kinds["object"]++
		for k, cv := range val {
			c.observe(n.child(k), cv)
		}

	case []interface{}:
		c.mark(n, len(val)) // cardinality lives on the array node
		n.kinds["array"]++
		for _, ev := range val {
			c.observeElem(n, ev)
		}

	default:
		c.observeScalar(n, v)
	}
}

// observeElem folds one array element into n. Object elements contribute their
// fields as n's children; scalar elements make n a leaf of that scalar type. n
// itself is not re-marked here — its presence/cardinality was set from the
// array length in observe.
func (c *census) observeElem(n *node, ev interface{}) {
	if obj, ok := ev.(map[string]interface{}); ok {
		n.kinds["object"]++
		for k, cv := range obj {
			c.observe(n.child(k), cv)
		}
		return
	}
	c.observeScalar(n, ev)
}

func (c *census) observeScalar(n *node, v interface{}) {
	c.mark(n, 1)

	var s string
	switch val := v.(type) {
	case string:
		n.kinds["string"]++
		s = val
	case json.Number:
		n.kinds["number"]++
		s = val.String()
	case bool:
		n.kinds["bool"]++
		if val {
			s = "true"
		} else {
			s = "false"
		}
	case nil:
		n.kinds["null"]++
		return
	default:
		n.kinds[fmt.Sprintf("%T", v)]++
		s = fmt.Sprintf("%v", v)
	}

	n.hasLeaf = true
	if n.sample == "" {
		n.sample = s
	}
	if len(n.values) < 60 {
		n.values[s]++
	}
}

// mark notes node n present in the current flight, remembering the largest
// count seen (array length, or 1 for objects and scalars).
func (c *census) mark(n *node, count int) {
	if count < 1 {
		count = 1
	}
	if count > c.seen[n] {
		c.seen[n] = count
	}
}

func (c *census) flush() {
	for n, count := range c.seen {
		n.flights++
		if count > n.maxRepeat {
			n.maxRepeat = count
		}
	}
	c.seen = map[*node]int{}
}

// ---------- rendering ----------

func (c *census) report(w io.Writer) {
	fmt.Fprintf(w, "NORMALIZED (fixm.normalized) observed shape\n")
	fmt.Fprintf(w, "envelopes: %d    flights: %d\n\n", c.envelopes, c.flights)
	fmt.Fprintf(w, "%s\n", strings.Repeat("=", 100))
	fmt.Fprintf(w, "  pct     = share of flights carrying this field\n")
	fmt.Fprintf(w, "  >       = nesting depth below a flight; no arrows means a top-level field\n")
	fmt.Fprintf(w, "  [xN]    = seen up to N times in one flight, so it is an array not a single value\n")
	fmt.Fprintf(w, "  :type   = JSON kind(s); a []prefix means an array of that element type\n")
	fmt.Fprintf(w, "  *       = full value set listed in the ENUMERATED section at the end\n")
	fmt.Fprintf(w, "%s\n\n", strings.Repeat("=", 100))

	c.render(w, c.root, 0)

	fmt.Fprintf(w, "\n\n%s\n", strings.Repeat("=", 100))
	fmt.Fprintf(w, "ENUMERATED VALUE SETS\n")
	fmt.Fprintf(w, "%s\n", strings.Repeat("=", 100))
	c.renderEnums(w, c.root, "")
}

func (c *census) render(w io.Writer, n *node, depth int) {
	arrows := strings.Repeat(">", depth)
	indent := strings.Repeat("  ", depth)

	repeat := ""
	if n.maxRepeat > 1 {
		repeat = fmt.Sprintf(" [x%d]", n.maxRepeat)
	}

	enum := ""
	if n.hasLeaf && len(n.values) > 1 && len(n.values) < 40 {
		enum = " *"
	}

	sample := ""
	if n.hasLeaf {
		sample = "  = " + truncate(n.sample, 40)
	}

	fmt.Fprintf(w, "%6.2f%%  %-8s %s%s%s  :%s%s%s\n",
		c.pct(n.flights), arrows, indent, n.name, repeat,
		typeLabel(n), enum, sample)

	for _, name := range n.order {
		c.render(w, n.children[name], depth+1)
	}
}

func (c *census) renderEnums(w io.Writer, n *node, path string) {
	here := n.name
	if path != "" {
		here = path + "/" + n.name
	}

	if n.hasLeaf && len(n.values) > 1 && len(n.values) < 40 {
		fmt.Fprintf(w, "\n%s   (%d distinct)\n", here, len(n.values))

		type kv struct {
			v string
			n int
		}
		var all []kv
		for v, cnt := range n.values {
			all = append(all, kv{v, cnt})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })

		for i, e := range all {
			if i >= 30 {
				fmt.Fprintf(w, "      ... %d more\n", len(all)-30)
				break
			}
			label := e.v
			if label == "" {
				label = "(empty)"
			}
			fmt.Fprintf(w, "      %-30s %8d\n", truncate(label, 30), e.n)
		}
	}

	for _, name := range n.order {
		c.renderEnums(w, n.children[name], here)
	}
}

func (c *census) pct(n int) float64 {
	if c.flights == 0 {
		return 0
	}
	return float64(n) / float64(c.flights) * 100
}

// typeLabel summarizes the JSON kinds seen at a node. An array of objects reads
// as "[]object", a scalar that is sometimes null as "string|null", and so on.
func typeLabel(n *node) string {
	var elem []string
	for _, k := range []string{"object", "string", "number", "bool", "null"} {
		if n.kinds[k] > 0 {
			elem = append(elem, k)
		}
	}
	label := strings.Join(elem, "|")

	if n.kinds["array"] > 0 {
		if label == "" {
			label = "any"
		}
		return "[]" + label
	}
	if label == "" {
		return "?"
	}
	return label
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// ---------- main ----------

func main() {
	brokers := flag.String("brokers", envOr("KAFKA_BROKERS", "localhost:9092"), "comma-separated broker list")
	topic := flag.String("topic", envOr("KAFKA_TOPIC_NORMALIZED", "fixm.normalized"), "topic to read")
	limit := flag.Int("limit", 0, "stop after N envelopes, 0 for all")
	idle := flag.Duration("idle", 5*time.Second, "stop after this long with no new messages")
	out := flag.String("out", "", "write report to this file instead of stdout")
	flag.Parse()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   strings.Split(*brokers, ","),
		Topic:     *topic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10 << 20,
	})
	defer reader.Close()

	if err := reader.SetOffset(kafka.FirstOffset); err != nil {
		log.Fatalf("seek: %v", err)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	c := newCensus()
	log.Printf("reading %s from %s", *topic, *brokers)

	for {
		if *limit > 0 && c.envelopes >= *limit {
			break
		}

		readCtx, readCancel := context.WithTimeout(ctx, *idle)
		msg, err := reader.ReadMessage(readCtx)
		readCancel()

		if err != nil {
			if ctx.Err() != nil {
				log.Println("interrupted")
			} else {
				log.Printf("done reading: %v", err)
			}
			break
		}

		c.envelopes++
		if err := c.walk(msg.Value); err != nil {
			log.Printf("walk failed at offset %d: %v", msg.Offset, err)
		}

		if c.envelopes%2000 == 0 {
			log.Printf("%d envelopes, %d flights", c.envelopes, c.flights)
		}
	}

	w := io.Writer(os.Stdout)
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			log.Fatalf("create %s: %v", *out, err)
		}
		defer f.Close()
		w = f
		log.Printf("writing report to %s", *out)
	}

	c.report(w)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
