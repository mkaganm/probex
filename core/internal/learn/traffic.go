package learn

import (
	"sort"
	"strings"
)

// TrafficAnalysis holds the results of analyzing traffic patterns.
type TrafficAnalysis struct {
	// Frequency maps each endpoint key to its call count.
	Frequency map[string]int `json:"frequency"`
	// CallSequences contains observed ordered sequences of endpoint calls.
	CallSequences []CallSequence `json:"call_sequences"`
	// Relationships contains detected endpoint relationships.
	Relationships []EndpointRelationship `json:"relationships"`
}

// CallSequence is an ordered group of endpoint calls that occur together.
type CallSequence struct {
	Endpoints []string `json:"endpoints"`
	Count     int      `json:"count"`
}

// RelationType describes the kind of relationship between two endpoints.
type RelationType string

const (
	// RelFollowedBy means endpoint A is typically followed by endpoint B.
	RelFollowedBy RelationType = "followed_by"
	// RelCreateThenRead means a POST is followed by a GET on the same resource.
	RelCreateThenRead RelationType = "create_then_read"
)

// EndpointRelationship describes a detected relationship between two endpoints.
type EndpointRelationship struct {
	From string       `json:"from"`
	To   string       `json:"to"`
	Type RelationType `json:"type"`
	// Count is how many times this relationship was observed.
	Count int `json:"count"`
}

// AnalyzeTraffic analyzes HAR entries for traffic patterns.
func AnalyzeTraffic(parsed *ParsedHAR) *TrafficAnalysis {
	analysis := &TrafficAnalysis{
		Frequency: make(map[string]int),
	}

	// Calculate frequency per endpoint.
	for key, entries := range parsed.Grouped {
		analysis.Frequency[key.String()] = len(entries)
	}

	// Analyze call ordering from chronological entries.
	if len(parsed.Ordered) > 1 {
		analysis.CallSequences = detectSequences(parsed.Ordered)
		analysis.Relationships = detectRelationships(parsed.Ordered)
	}

	return analysis
}

// detectSequences finds repeated sequences of endpoint calls using bigram analysis.
func detectSequences(entries []Entry) []CallSequence {
	// Build sequence of endpoint keys.
	keys := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = entryToKey(e).String()
	}

	// Count bigrams (pairs of consecutive calls).
	bigramCounts := make(map[[2]string]int)
	for i := 0; i < len(keys)-1; i++ {
		pair := [2]string{keys[i], keys[i+1]}
		bigramCounts[pair]++
	}

	// Count trigrams (triples of consecutive calls).
	trigramCounts := make(map[[3]string]int)
	for i := 0; i < len(keys)-2; i++ {
		triple := [3]string{keys[i], keys[i+1], keys[i+2]}
		trigramCounts[triple]++
	}

	var sequences []CallSequence

	// Add bigrams that appear more than once.
	for pair, count := range bigramCounts {
		if count >= 2 {
			sequences = append(sequences, CallSequence{
				Endpoints: []string{pair[0], pair[1]},
				Count:     count,
			})
		}
	}

	// Add trigrams that appear more than once.
	for triple, count := range trigramCounts {
		if count >= 2 {
			sequences = append(sequences, CallSequence{
				Endpoints: []string{triple[0], triple[1], triple[2]},
				Count:     count,
			})
		}
	}

	// Sort by count descending.
	sort.Slice(sequences, func(i, j int) bool {
		return sequences[i].Count > sequences[j].Count
	})

	return sequences
}

// detectRelationships identifies semantic relationships between endpoints.
func detectRelationships(entries []Entry) []EndpointRelationship {
	pairCounts := make(map[[2]string]int)

	for i := 0; i < len(entries)-1; i++ {
		from := entryToKey(entries[i])
		to := entryToKey(entries[i+1])
		pair := [2]string{from.String(), to.String()}
		pairCounts[pair]++
	}

	var rels []EndpointRelationship
	seen := make(map[[2]string]bool)

	for i := 0; i < len(entries)-1; i++ {
		from := entryToKey(entries[i])
		to := entryToKey(entries[i+1])
		pair := [2]string{from.String(), to.String()}

		if seen[pair] {
			continue
		}
		seen[pair] = true

		count := pairCounts[pair]
		if count < 2 {
			continue
		}

		relType := RelFollowedBy

		// Detect create-then-read: POST /resource followed by GET /resource/{id}.
		if from.Method == "POST" && to.Method == "GET" {
			fromBase := strings.TrimSuffix(from.Path, "/")
			toBase := strings.TrimSuffix(to.Path, "/")
			// Check if the GET path is the POST path + /{id}.
			if strings.HasPrefix(toBase, fromBase+"/") {
				relType = RelCreateThenRead
			}
		}

		rels = append(rels, EndpointRelationship{
			From:  from.String(),
			To:    to.String(),
			Type:  relType,
			Count: count,
		})
	}

	// Sort by count descending.
	sort.Slice(rels, func(i, j int) bool {
		return rels[i].Count > rels[j].Count
	})

	return rels
}
