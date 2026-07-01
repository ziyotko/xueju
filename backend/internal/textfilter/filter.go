package textfilter

import (
	_ "embed"
	"sort"
	"strings"
)

//go:embed sensitive_words.txt
var defaultWords string

type word struct {
	text  string
	runes []rune
}

type Filter struct {
	words []word
}

func New() (*Filter, error) {
	words := parseWords(defaultWords)
	return &Filter{words: words}, nil
}

func (f *Filter) Clean(text string) string {
	if f == nil || len(f.words) == 0 || text == "" {
		return text
	}

	source := []rune(text)
	lower := []rune(strings.ToLower(text))
	var builder strings.Builder
	for i := 0; i < len(source); {
		if matched := f.matchAt(lower, i); matched > 0 {
			builder.WriteString(strings.Repeat("*", matched))
			i += matched
			continue
		}
		builder.WriteRune(source[i])
		i++
	}
	return builder.String()
}

func (f *Filter) CleanSlice(values []string) []string {
	if len(values) == 0 {
		return values
	}
	cleaned := make([]string, len(values))
	for i, value := range values {
		cleaned[i] = f.Clean(value)
	}
	return cleaned
}

func (f *Filter) matchAt(text []rune, start int) int {
	for _, item := range f.words {
		if start+len(item.runes) > len(text) {
			continue
		}
		ok := true
		for offset, r := range item.runes {
			if text[start+offset] != r {
				ok = false
				break
			}
		}
		if ok {
			return len(item.runes)
		}
	}
	return 0
}

func parseWords(content string) []word {
	seen := map[string]struct{}{}
	words := []word{}
	for _, line := range strings.Split(content, "\n") {
		text := strings.TrimSpace(line)
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		text = strings.ToLower(text)
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		words = append(words, word{text: text, runes: []rune(text)})
	}
	sort.Slice(words, func(i, j int) bool {
		if len(words[i].runes) == len(words[j].runes) {
			return words[i].text < words[j].text
		}
		return len(words[i].runes) > len(words[j].runes)
	})
	return words
}
