// Package polygon parses Ticketmaster venue map SVGs, extracting the vertices
// of each section's polygon (a <path data-component="svg__section"> element)
// alongside the section name/id used to correlate it with seat data elsewhere
// in this repo.
package polygon

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
)

const sectionComponent = "svg__section"

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Section struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Points []Point `json:"points"`
}

// ParseFile opens and parses the SVG file at path. See Parse for details.
func ParseFile(path string) ([]Section, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening svg file: %w", err)
	}
	defer f.Close()

	sections, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return sections, nil
}

// Parse reads an SVG document and returns one Section per
// <path data-component="svg__section"> element, in document order, with the
// polygon vertices decoded from its "d" attribute.
func Parse(r io.Reader) ([]Section, error) {
	decoder := xml.NewDecoder(r)

	var sections []Section
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading xml token: %w", err)
		}

		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "path" {
			continue
		}

		attrs := attrMap(start.Attr)
		if attrs["data-component"] != sectionComponent {
			continue
		}

		points, err := parsePathData(attrs["d"])
		if err != nil {
			return nil, fmt.Errorf("section %q: %w", attrs["data-section-name"], err)
		}

		sections = append(sections, Section{
			ID:     attrs["data-section-id"],
			Name:   attrs["data-section-name"],
			Points: points,
		})
	}
	return sections, nil
}

func attrMap(attrs []xml.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Name.Local] = a.Value
	}
	return m
}

// pathTokenRe matches the absolute path commands (M, L, C, Z) and numbers
// (including decimals and negatives) found in these venue maps' "d"
// attributes. Other SVG path commands (arcs, relative commands, quadratic
// curves, implicit command repetition) are not present in the source data
// and are intentionally unsupported.
var pathTokenRe = regexp.MustCompile(`[MLCZ]|-?\d+(?:\.\d+)?`)

// parsePathData decodes a path "d" attribute into its constituent vertices.
// M and L each contribute one point; C (a cubic bezier) contributes its two
// control points followed by its endpoint, so the returned points trace the
// polygon's outline without approximating the curve itself. Z is a no-op
// since it closes the path back to the first point rather than adding one.
func parsePathData(d string) ([]Point, error) {
	tokens := pathTokenRe.FindAllString(d, -1)

	var points []Point
	var cmd byte
	i := 0
	for i < len(tokens) {
		tok := tokens[i]
		if len(tok) == 1 && (tok[0] == 'M' || tok[0] == 'L' || tok[0] == 'C' || tok[0] == 'Z') {
			cmd = tok[0]
			i++
			continue
		}

		switch cmd {
		case 'M', 'L':
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("incomplete coordinate pair for command %q in %q", cmd, d)
			}
			p, err := parsePoint(tokens[i], tokens[i+1])
			if err != nil {
				return nil, err
			}
			points = append(points, p)
			i += 2
		case 'C':
			if i+5 >= len(tokens) {
				return nil, fmt.Errorf("incomplete curve arguments for command %q in %q", cmd, d)
			}
			for j := 0; j < 3; j++ {
				p, err := parsePoint(tokens[i+j*2], tokens[i+j*2+1])
				if err != nil {
					return nil, err
				}
				points = append(points, p)
			}
			i += 6
		default:
			return nil, fmt.Errorf("number %q precedes any recognized command in %q", tok, d)
		}
	}
	return points, nil
}

func parsePoint(xTok, yTok string) (Point, error) {
	x, err := strconv.ParseFloat(xTok, 64)
	if err != nil {
		return Point{}, fmt.Errorf("parsing x coordinate %q: %w", xTok, err)
	}
	y, err := strconv.ParseFloat(yTok, 64)
	if err != nil {
		return Point{}, fmt.Errorf("parsing y coordinate %q: %w", yTok, err)
	}
	return Point{X: x, Y: y}, nil
}
