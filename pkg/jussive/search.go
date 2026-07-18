package jussive

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

type SearchResult struct {
	ID      string   `json:"id" yaml:"id"`
	Name    string   `json:"name" yaml:"name"`
	Summary string   `json:"summary" yaml:"summary"`
	Tags    []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Score   float64  `json:"score" yaml:"score"`
}

func SearchMetadata(metadata []CommandMetadata, query string, limit int) []SearchResult {
	if limit <= 0 {
		limit = 5
	}
	queryTokens := tokenize(query)
	var results []SearchResult
	for _, m := range metadata {
		score := scoreMetadata(m, queryTokens)
		if score <= 0 {
			continue
		}
		results = append(results, SearchResult{
			ID:      m.ID,
			Name:    m.Name,
			Summary: m.Summary,
			Tags:    m.Tags,
			Score:   math.Round(score*100) / 100,
		})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].ID < results[j].ID
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		return results[:limit]
	}
	return results
}

func scoreMetadata(m CommandMetadata, query []string) float64 {
	if len(query) == 0 {
		return 0
	}
	fields := []struct {
		text   string
		weight float64
	}{
		{m.ID, 5},
		{m.Name, 4},
		{m.Summary, 3},
		{strings.Join(m.Tags, " "), 4},
		{strings.Join(m.Examples, " "), 2},
		{strings.Join(m.WhenToUse, " "), 2},
		{strings.Join(m.WhenNotToUse, " "), -1},
	}
	for _, input := range m.Inputs {
		fields = append(fields, struct {
			text   string
			weight float64
		}{input.Name + " " + input.Description, 1.5})
	}
	var total float64
	for _, token := range query {
		for _, field := range fields {
			for _, candidate := range tokenize(field.text) {
				switch {
				case candidate == token:
					total += field.weight
				case strings.Contains(candidate, token) || strings.Contains(token, candidate):
					total += field.weight * 0.4
				}
			}
		}
	}
	return total
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	return strings.FieldsFunc(text, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	})
}
