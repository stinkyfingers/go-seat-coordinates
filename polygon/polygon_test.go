package polygon

import (
	"strings"
	"testing"
)

func TestParsePathData(t *testing.T) {
	tests := []struct {
		name    string
		d       string
		want    []Point
		wantErr bool
	}{
		{
			name: "move and lines",
			d:    "M3831.48,680.05L3627.42,766.89L3483.2,868.684Z",
			want: []Point{
				{3831.48, 680.05},
				{3627.42, 766.89},
				{3483.2, 868.684},
			},
		},
		{
			name: "negative coordinates",
			d:    "M-10.5,-20L30,-40Z",
			want: []Point{
				{-10.5, -20},
				{30, -40},
			},
		},
		{
			name: "move line and curve",
			d:    "M4093.1,4921.9L3920.35,5073.67C4137.09,5323.51,4136.01,5321.79,4136.01,5321.79Z",
			want: []Point{
				{4093.1, 4921.9},
				{3920.35, 5073.67},
				{4137.09, 5323.51},
				{4136.01, 5321.79},
				{4136.01, 5321.79},
			},
		},
		{
			name:    "incomplete pair",
			d:       "M100,200L300",
			wantErr: true,
		},
		{
			name:    "incomplete curve",
			d:       "M0,0C1,1,2,2",
			wantErr: true,
		},
		{
			name:    "number before any command",
			d:       "100,200",
			wantErr: true,
		},
		{
			name: "empty",
			d:    "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePathData(tt.d)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parsePathData(%q) expected error, got none", tt.d)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePathData(%q) unexpected error: %v", tt.d, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parsePathData(%q) = %v, want %v", tt.d, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("point %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

const sampleSVG = `<svg viewBox="0 0 100 100">
  <g class="polygons">
    <path data-component="svg__section" data-section-id="s_1" data-section-name="101"
      class="block"
      d="M0,0L10,0L10,10L0,10Z"></path>
    <path data-component="svg__section" data-section-id="s_2" data-section-name="102 TABLES"
      class="block is-ga"
      d="M20,20L30,20L30,30Z">
    </path>
    <path data-component="not-a-section" data-section-id="s_3" data-section-name="ignored"
      d="M99,99L98,98Z"></path>
  </g>
  <g class="labels">
    <text data-component="svg__label" data-label-id="s_1" class="label"><tspan>101</tspan></text>
  </g>
</svg>`

func TestParse(t *testing.T) {
	sections, err := Parse(strings.NewReader(sampleSVG))
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if len(sections) != 2 {
		t.Fatalf("Parse() returned %d sections, want 2 (non-section paths and labels must be excluded)", len(sections))
	}

	first := sections[0]
	if first.ID != "s_1" || first.Name != "101" {
		t.Errorf("first section = %+v, want ID s_1, Name 101", first)
	}
	wantPoints := []Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	if len(first.Points) != len(wantPoints) {
		t.Fatalf("first section points = %v, want %v", first.Points, wantPoints)
	}
	for i, p := range wantPoints {
		if first.Points[i] != p {
			t.Errorf("first section point %d = %v, want %v", i, first.Points[i], p)
		}
	}

	second := sections[1]
	if second.ID != "s_2" || second.Name != "102 TABLES" {
		t.Errorf("second section = %+v, want ID s_2, Name '102 TABLES'", second)
	}
}

// TestParseFile_RealVenueSVG parses an actual downloaded venue map to guard
// against the assumptions above (only M/L/C/Z, always exactly one
// data-section-name per svg__section path) drifting from the real data.
func TestParseFile_RealVenueSVG(t *testing.T) {
	sections, err := ParseFile("../svgs/atb.svg")
	if err != nil {
		t.Fatalf("ParseFile() unexpected error: %v", err)
	}

	const wantCount = 254
	if len(sections) != wantCount {
		t.Fatalf("ParseFile() returned %d sections, want %d", len(sections), wantCount)
	}

	for _, s := range sections {
		if s.Name == "" {
			t.Errorf("section %q has empty Name", s.ID)
		}
		if len(s.Points) < 3 {
			t.Errorf("section %q (%s) has %d points, want a closed polygon (>=3)", s.Name, s.ID, len(s.Points))
		}
	}

	first := sections[0]
	if first.ID != "s_370" || first.Name != "148" {
		t.Fatalf("first section = %+v, want ID s_370, Name 148", first)
	}
	if first.Points[0] != (Point{X: 3831.48, Y: 680.05}) {
		t.Errorf("first section's first point = %v, want {3831.48 680.05}", first.Points[0])
	}
}
