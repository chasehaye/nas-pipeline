// Command census reconstructs the shape of the FIXM feed from the data
// itself and prints it as a JSON-like nested tree.
//
// A flat list of paths tells you what exists but makes you rebuild the
// hierarchy mentally. This renders what the XML actually looks like:
// nesting, presence, cardinality, and a real sample value for every leaf,
// so the output can be read straight down while writing structs.
//
// Reads the partition directly rather than joining a consumer group, so it
// always starts from the beginning and never disturbs the processor's
// offsets. Safe to run repeatedly.
package main

import (
	"context"
	"encoding/xml"
	"errors"
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

// Attributes worth enumerating in full. Anything not listed still gets a
// sample value and a cardinality count, just not the complete value set.
// High-cardinality fields like callsigns would be a memory leak and tell
// you nothing.
var enumerate = map[string]bool{
	"source": true, "system": true, "flightType": true,
	"fdpsFlightStatus": true, "reportSource": true, "coastIndicator": true,
	"event": true, "wakeTurbulence": true, "equipmentQualifier": true,
	"uom": true, "codeSpace": true, "invalid": true, "nil": true,
	"initialFlightRules": true, "airspaceType": true, "centre": true,
	"coordinationTimeHandling": true, "standardCapabilities": true,
	"tfmsSpecialAircraftQualifier": true, "aircraftPerformance": true,
	"phase": true, "airborneHold": true,
}

// node is one element in the reconstructed tree
type node struct {
	name string

	// flights this element appeared in at all
	flights int

	// how many times it appeared within a single flight, at most
	maxRepeat int

	// leaf text content, if any
	hasText    bool
	sampleText string
	textValues map[string]int

	attrs    map[string]*attr
	children map[string]*node

	// insertion order, so siblings print in the order first seen rather
	// than alphabetically -- closer to how the document reads
	order []string
}

type attr struct {
	name    string
	flights int
	sample  string
	values  map[string]int
}

func newNode(name string) *node {
	return &node{
		name:       name,
		attrs:      map[string]*attr{},
		children:   map[string]*node{},
		textValues: map[string]int{},
	}
}

type census struct {
	envelopes int
	flights   int
	root      *node
	seen      map[*node]int
	seenAt    map[*attr]bool
}

func newCensus() *census {
	return &census{
		root:   newNode("flight"),
		seen:   map[*node]int{},
		seenAt: map[*attr]bool{},
	}
}

func (c *census) walk(data []byte) error {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var stack []*node
	inFlight := false

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		switch t := tok.(type) {

		case xml.StartElement:
			if t.Name.Local == "flight" {
				c.flights++
				c.flush()
				inFlight = true
				stack = []*node{c.root}
				c.record(c.root, t.Attr)
				continue
			}
			if !inFlight {
				continue
			}

			parent := stack[len(stack)-1]
			n, ok := parent.children[t.Name.Local]
			if !ok {
				n = newNode(t.Name.Local)
				parent.children[t.Name.Local] = n
				parent.order = append(parent.order, t.Name.Local)
			}
			stack = append(stack, n)
			c.record(n, t.Attr)

		case xml.CharData:
			if !inFlight || len(stack) == 0 {
				continue
			}
			text := strings.TrimSpace(string(t))
			if text == "" {
				continue
			}
			n := stack[len(stack)-1]
			n.hasText = true
			if n.sampleText == "" {
				n.sampleText = text
			}
			if len(n.textValues) < 60 {
				n.textValues[text]++
			}

		case xml.EndElement:
			if t.Name.Local == "flight" {
				inFlight = false
				stack = nil
				continue
			}
			if inFlight && len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
}

func (c *census) record(n *node, attrs []xml.Attr) {
	c.seen[n]++

	for _, a := range attrs {
		if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
			continue
		}
		if a.Name.Local == "type" && a.Name.Space != "" {
			continue
		}

		at, ok := n.attrs[a.Name.Local]
		if !ok {
			at = &attr{name: a.Name.Local}
			if enumerate[a.Name.Local] {
				at.values = map[string]int{}
			}
			n.attrs[a.Name.Local] = at
		}
		if at.sample == "" {
			at.sample = a.Value
		}
		if at.values != nil && len(at.values) < 300 {
			at.values[a.Value]++
		}
		c.seenAt[at] = true
	}
}

func (c *census) flush() {
	for n, count := range c.seen {
		n.flights++
		if count > n.maxRepeat {
			n.maxRepeat = count
		}
	}
	for a := range c.seenAt {
		a.flights++
	}
	c.seen = map[*node]int{}
	c.seenAt = map[*attr]bool{}
}

// ---------- rendering ----------

func (c *census) report(w io.Writer, showPct, color bool) {
	fmt.Fprintf(w, "// FIXM observed shape  (envelopes: %d, flights: %d)\n", c.envelopes, c.flights)
	fmt.Fprintf(w, "// nested like JSON: <element> { ... }, leaves show a sample value.\n")
	fmt.Fprintf(w, "//   @name  = XML attribute (struct tag `,attr`)\n")
	fmt.Fprintf(w, "//   #text  = the element's own text when it also has attributes/children\n")
	fmt.Fprintf(w, "//   [xN]   = seen up to N times in one flight, so it is a slice not a single value\n")
	fmt.Fprintf(w, "//   *      = full value set listed in the ENUMERATED section at the end\n")
	if showPct {
		fmt.Fprintf(w, "//   // NN%% = share of flights carrying the field (-pct)\n")
	}
	fmt.Fprintf(w, "\n")

	c.render(w, c.root, 0, showPct, color)

	fmt.Fprintf(w, "\n%s\n", strings.Repeat("=", 100))
	fmt.Fprintf(w, "ENUMERATED VALUE SETS\n")
	fmt.Fprintf(w, "%s\n", strings.Repeat("=", 100))
	c.renderEnums(w, c.root, "")
}

func (c *census) render(w io.Writer, n *node, depth int, showPct, color bool) {
	ind := strings.Repeat("  ", depth)

	repeat := ""
	if n.maxRepeat > 1 {
		repeat = fmt.Sprintf(" [x%d]", n.maxRepeat)
	}

	attrNames := sortedAttrNames(n)
	isContainer := len(n.children) > 0 || len(attrNames) > 0

	// A pure leaf renders as   name: "sample"
	if !isContainer {
		line := fmt.Sprintf("%s%s%s: %s%s%s",
			ind, n.name, repeat, leafValue(n), enumMark(n), c.pctComment(n.flights, showPct))
		fmt.Fprintln(w, paint(depth, line, color))
		return
	}

	// A container opens a brace block:   name [xN] {
	fmt.Fprintln(w, paint(depth,
		fmt.Sprintf("%s%s%s {%s", ind, n.name, repeat, c.pctComment(n.flights, showPct)), color))

	// Attributes first, most common first, marked with @ (they map to `,attr`).
	// They belong to this element, so they share its depth color.
	for _, k := range attrNames {
		a := n.attrs[k]
		mark := ""
		if a.values != nil {
			mark = " *"
		}
		line := fmt.Sprintf("%s  @%s: %s%s%s",
			ind, a.name, quote(a.sample), mark, c.pctComment(a.flights, showPct))
		fmt.Fprintln(w, paint(depth, line, color))
	}

	// Mixed content: element has attributes/children but also its own text.
	if n.hasText {
		fmt.Fprintln(w, paint(depth, fmt.Sprintf("%s  #text: %s", ind, quote(n.sampleText)), color))
	}

	for _, name := range n.order {
		c.render(w, n.children[name], depth+1, showPct, color)
	}

	fmt.Fprintln(w, paint(depth, fmt.Sprintf("%s}", ind), color))
}

// depthColors is a red -> orange -> yellow -> green -> cyan -> blue -> violet
// ramp in 256-color ANSI, indexed by nesting depth (cycles when deeper than
// the ramp). Each nesting level gets its own band so the braces read at a glance.
var depthColors = []int{196, 202, 208, 214, 220, 226, 190, 154, 118, 82, 48, 51, 45, 39, 33, 99, 129, 165, 201}

func paint(depth int, s string, on bool) string {
	if !on {
		return s
	}
	code := depthColors[depth%len(depthColors)]
	return fmt.Sprintf("\x1b[38;5;%dm%s\x1b[0m", code, s)
}

func sortedAttrNames(n *node) []string {
	var names []string
	for k := range n.attrs {
		names = append(names, k)
	}
	sort.Slice(names, func(i, j int) bool {
		return n.attrs[names[i]].flights > n.attrs[names[j]].flights
	})
	return names
}

// leafValue is the sample text of a leaf element, or "" for an empty element
// (the XML equivalent of a JSON {} / null field).
func leafValue(n *node) string {
	if !n.hasText {
		return `""`
	}
	return quote(n.sampleText)
}

func quote(s string) string {
	return `"` + truncate(s, 60) + `"`
}

// enumMark flags a leaf whose text has a small, fixed value set, pointing the
// reader at the ENUMERATED section (same * meaning as on attributes).
func enumMark(n *node) string {
	if n.hasText && len(n.textValues) > 1 && len(n.textValues) < 40 {
		return " *"
	}
	return ""
}

func (c *census) pctComment(flights int, show bool) string {
	if !show {
		return ""
	}
	return fmt.Sprintf("   // %.0f%%", c.pct(flights))
}

func (c *census) renderEnums(w io.Writer, n *node, path string) {
	here := path
	if here == "" {
		here = n.name
	} else {
		here = path + "/" + n.name
	}

	var names []string
	for k, a := range n.attrs {
		if len(a.values) > 0 {
			names = append(names, k)
		}
	}
	sort.Strings(names)

	for _, k := range names {
		a := n.attrs[k]
		fmt.Fprintf(w, "\n%s/@%s   (%d distinct)\n", here, k, len(a.values))

		type kv struct {
			v string
			n int
		}
		var all []kv
		for v, cnt := range a.values {
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
			fmt.Fprintf(w, "      %-30s %8d\n", label, e.n)
		}
	}

	if n.hasText && len(n.textValues) > 1 && len(n.textValues) < 40 {
		fmt.Fprintf(w, "\n%s   (text, %d distinct)\n", here, len(n.textValues))
		type kv struct {
			v string
			n int
		}
		var all []kv
		for v, cnt := range n.textValues {
			all = append(all, kv{v, cnt})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })
		for i, e := range all {
			if i >= 20 {
				fmt.Fprintf(w, "      ... %d more\n", len(all)-20)
				break
			}
			fmt.Fprintf(w, "      %-30s %8d\n", truncate(e.v, 30), e.n)
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
	topic := flag.String("topic", envOr("KAFKA_TOPIC_RAW", "fixm.raw"), "topic to read")
	limit := flag.Int("limit", 0, "stop after N envelopes, 0 for all")
	idle := flag.Duration("idle", 5*time.Second, "stop after this long with no new messages")
	out := flag.String("out", "", "write report to this file instead of stdout")
	pct := flag.Bool("pct", false, "annotate every field with the % of flights carrying it")
	color := flag.Bool("color", true, "color each nesting depth (auto-off when -out writes a file)")
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

	c.flush()

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

	// Color would corrupt a file; keep it terminal-only.
	c.report(w, *pct, *color && *out == "")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}