package mapping

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/stinkyfingers/go-seat-coordinates/seat"
	"github.com/stinkyfingers/go-seat-coordinates/seat/external_seat"
	"github.com/stinkyfingers/go-seat-coordinates/seat/internal_seat"
)

const (
	bedrockModelID         = "us.amazon.nova-lite-v1:0"
	bedrockRegion          = "us-west-1"
	bedrockMaxOutputTokens = 4000
)

// sectionSuggestion is the shape the model is instructed to return for each unmapped internal section.
type sectionSuggestion struct {
	InternalSection        string  `json:"internal_section"`
	MatchedExternalSection *string `json:"matched_external_section"`
	Confidence             string  `json:"confidence"`
	Reason                 string  `json:"reason"`
}

var normalizeRe = regexp.MustCompile(`[^A-Z0-9]+`)

// MapInternalToExternal takes a list of internal seats and a list of external seats
// and returns a mapping of internal seat keys to external seat keys and unmappable internal sections.
func MapInternalToExternal(externalSeats, internalSeats seat.Seaters, skipLLM bool) ([]seat.Seat, []string, error) {
	internalSections := uniqueSections(internalSeats)
	externalSections := uniqueSections(externalSeats)
	sectionMap, unmappedInternalSections, unmappedExternalSections := getPerfectSectionMatches(externalSections, internalSections)
	fmt.Printf("perfectly mapped %d sections, leaving %d unmapped internal sections and %d unmapped external sections\n", len(sectionMap), len(unmappedInternalSections), len(unmappedExternalSections))

	// LLM is not skipped and there are unmapped sections, so attempt to map them using LLM
	if !skipLLM && len(unmappedExternalSections) > 0 && len(unmappedInternalSections) > 0 {
		fmt.Printf("using LLM to attempt to map %d sections to %d unused external sections\n", len(unmappedInternalSections), len(unmappedExternalSections))
		aiSectionMap, err := aiMapSections(unmappedExternalSections, unmappedInternalSections)
		if err != nil {
			return nil, nil, fmt.Errorf("mapping sections with AI: %w", err)
		}

		// Merge the two maps
		for k, v := range aiSectionMap {
			sectionMap[k] = v
		}
	}

	missingSectionsTally := make(map[string]struct{})

	// populate seats
	var seats []seat.Seat
	for _, internalSeat := range internalSeats {
		internalSection := internalSeat.GetSection()
		externalSectionName, ok := sectionMap[internalSection]
		if !ok {
			missingSectionsTally[internalSection] = struct{}{}
			continue
		}
		// Find the external seat that matches the internal seat's section, row, and seat number
		var matchedExternalSeat *external_seat.Seat
		found := false
		for _, externalSeat := range externalSeats {
			if externalSeat.GetSection() == externalSectionName &&
				externalSeat.GetRow() == internalSeat.GetRow() &&
				externalSeat.GetSeatNumber() == internalSeat.GetSeatNumber() {
				matchedExternalSeat = externalSeat.(*external_seat.Seat)
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("no matching external seat found for internal seat %s:%s:%s\n", internalSection, internalSeat.GetRow(), internalSeat.GetSeatNumber())
			continue
		}
		seats = append(seats, seat.Seat{
			InternalID:  internalSeat.(*internal_seat.Seat).SeatID,
			SeatSection: internalSeat.GetSection(),
			SeatRow:     internalSeat.GetRow(),
			SeatNumber:  internalSeat.GetSeatNumber(),
			X:           matchedExternalSeat.X,
			Y:           matchedExternalSeat.Y,
		})
	}

	var missingSections []string
	for section := range missingSectionsTally {
		missingSections = append(missingSections, section)
	}
	return seats, missingSections, nil
}

func uniqueSections(seaters []seat.Seater) []string {
	unique := make(map[string]struct{})
	for _, seater := range seaters {
		section := seater.GetSection()
		if _, exists := unique[section]; !exists {
			unique[section] = struct{}{}
		}
	}
	sections := make([]string, 0, len(unique))
	for section := range unique {
		sections = append(sections, section)
	}
	return sections
}

// getPerfectSectionMatches returns a map of internal section names: external section names where the names match exactly.
// It also returns slices of unmatched internal and external section names.
func getPerfectSectionMatches(externalSections, internalSections []string) (map[string]string, []string, []string) {
	matches := make(map[string]string)
	matchedExternal := make([]string, 0)
	for _, extSection := range externalSections {
		for _, intSection := range internalSections {
			if normalizeSection(extSection) == normalizeSection(intSection) {
				matches[intSection] = extSection
				matchedExternal = append(matchedExternal, extSection)
			}
		}
	}

	var unmatchedInternal []string
	var unmatchedExternal []string
	for _, intSection := range internalSections {
		if _, ok := matches[intSection]; !ok {
			unmatchedInternal = append(unmatchedInternal, intSection)
		}
	}
	for _, extSection := range externalSections {
		if _, ok := matches[extSection]; !ok {
			unmatchedExternal = append(unmatchedExternal, extSection)
		}
	}
	return matches, unmatchedInternal, unmatchedExternal
}

func normalizeSection(value string) string {
	return normalizeRe.ReplaceAllString(strings.ToUpper(value), "")
}

// aiMapSections returns a map of internal section names: external section names where the names do not match exactly, using AWS Bedrock to find the "best" match
func aiMapSections(externalSections, internalSections []string) (map[string]string, error) {
	ctx := context.Background()
	client, err := newBedrockClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating bedrock client: %w", err)
	}
	llmMatches, err := getLLMSectionMatches(ctx, client, externalSections, internalSections)
	if err != nil {
		return nil, fmt.Errorf("getting llm section matches: %w", err)
	}
	return llmMatches, nil
}

func chunkLLMRequest(externalSections, internalSections []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(internalSections); i += chunkSize {
		end := i + chunkSize
		if end > len(internalSections) {
			end = len(internalSections)
		}
		chunks = append(chunks, internalSections[i:end])
	}
	return chunks
}

// getLLMSectionMatches asks Bedrock, to suggest an external section for each
// unresolved internal section and returns a map of internal section name -> suggested
// external section name (empty when the model found no credible match).
func getLLMSectionMatches(ctx context.Context, client *bedrockruntime.Client, externalSections, internalSections []string) (map[string]string, error) {
	suggestions := make(map[string]string)

	chunkedInternalSections := chunkLLMRequest(externalSections, internalSections, 20)
	for _, internalChunk := range chunkedInternalSections {
		prompt := buildSectionPrompt(externalSections, internalChunk)
		// fmt.Println("prompt: ", prompt)

		raw, err := callBedrock(ctx, client, prompt)
		if err != nil {
			return nil, err
		}
		sectionSuggestions, err := extractSectionSuggestions(raw)
		if err != nil {
			return nil, fmt.Errorf("parsing bedrock response: %w", err)
		}
		for _, s := range sectionSuggestions {
			if s.MatchedExternalSection != nil && s.Confidence == "high" {
				suggestions[s.InternalSection] = *s.MatchedExternalSection
			}
		}
	}
	fmt.Println("AI section mapping suggestions:", suggestions)
	return suggestions, nil
}

func buildSectionPrompt(externalSections, internalSections []string) string {
	type internalSectionPayload struct {
		InternalSection string `json:"internal_section"`
	}
	type externalSectionPayload struct {
		ExternalSection string `json:"external_section"`
	}

	internalPayload := make([]internalSectionPayload, len(internalSections))
	for i, s := range internalSections {
		internalPayload[i] = internalSectionPayload{InternalSection: s}
	}
	externalPayload := make([]externalSectionPayload, len(externalSections))
	for i, s := range externalSections {
		externalPayload[i] = externalSectionPayload{ExternalSection: s}
	}

	internalJSON, _ := json.MarshalIndent(internalPayload, "", "  ")
	externalJSON, _ := json.MarshalIndent(externalPayload, "", "  ")

	instructions := strings.TrimSpace(`
You are mapping internal seat section names to external seat section records.

Return only a JSON array. Do not include markdown fences or extra text.
Each array item must have this shape:
{
  "internal_section": "original internal section name",
  "matched_external_section": "exact external_section value" | null,
  "confidence": "high" | "medium" | "low"
}

Rules:
- Use the exact internal section string provided.
- If there is a plausible external match, return the exact external_section string.
- Prefer semantic matches such as suite, club, patio, standing room, terrace, box, lounge, and numbered variants.
- If the best answer is uncertain, you may still provide a best-effort match with confidence "low".
- If there is no credible match, omit internal_section from the result.
- Do not invent external sections that are not in the external list.
- Watch for logical abbreviations, e.g. CL101 is a likely match for Club 101, and return the exact external_section string.
- Watch for situations where external sections omit a prefix, e.g. STE150 likely matches 150, and return the exact external_section string.
- Avoid partial numerical matches, e.g. 101 may match 101A, but 36 should not match 136 and LUX-26 should not match 226-WC.
`)

	return strings.Join([]string{
		instructions,
		fmt.Sprintf("Internal sections to map:\n%s", internalJSON),
		fmt.Sprintf("Available external sections:\n%s", externalJSON),
	}, "\n\n")
}

var jsonArrayRe = regexp.MustCompile(`(?s)\[.*\]`)
var codeFenceOpenRe = regexp.MustCompile("^```[a-zA-Z0-9_-]*\\s*")
var codeFenceCloseRe = regexp.MustCompile("\\s*```$")

func extractSectionSuggestions(text string) ([]sectionSuggestion, error) {
	cleaned := strings.TrimSpace(text)
	if strings.HasPrefix(cleaned, "```") {
		cleaned = codeFenceOpenRe.ReplaceAllString(cleaned, "")
		cleaned = codeFenceCloseRe.ReplaceAllString(cleaned, "")
	}

	var suggestions []sectionSuggestion
	if err := json.Unmarshal([]byte(cleaned), &suggestions); err == nil {
		return suggestions, nil
	}

	match := jsonArrayRe.FindString(cleaned)
	if match == "" {
		return nil, fmt.Errorf("bedrock response did not contain a JSON array: %s", text)
	}
	if err := json.Unmarshal([]byte(match), &suggestions); err != nil {
		return nil, fmt.Errorf("bedrock response JSON was not an array: %w", err)
	}
	return suggestions, nil
}

func newBedrockClient(ctx context.Context) (*bedrockruntime.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(bedrockRegion))
	if err != nil {
		return nil, err
	}
	return bedrockruntime.NewFromConfig(cfg), nil
}

func callBedrock(ctx context.Context, client *bedrockruntime.Client, prompt string) (string, error) {
	resp, err := client.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String(bedrockModelID),
		Messages: []types.Message{
			{
				Role:    types.ConversationRoleUser,
				Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: prompt}},
			},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens:   aws.Int32(int32(bedrockMaxOutputTokens)),
			Temperature: aws.Float32(0),
		},
	})
	if err != nil {
		return "", fmt.Errorf("calling bedrock: %w", err)
	}

	output, ok := resp.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return "", fmt.Errorf("unexpected bedrock output type: %T", resp.Output)
	}

	var text strings.Builder
	for _, block := range output.Value.Content {
		if textBlock, ok := block.(*types.ContentBlockMemberText); ok {
			text.WriteString(textBlock.Value)
		}
	}
	return text.String(), nil
}
