package main

import (
	"regexp"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

// FixEncoding tries to fix mojibake from UTF-8 misinterpreted as Windows-1252
func FixEncoding(input string) string {
	decoder := charmap.Windows1252.NewDecoder()
	out, err := decoder.String(input)
	if err != nil {
		return input
	}
	return out
}

// Digraph-based phonetic map
var digraphs = map[string]string{
	"sh": "ш", "ch": "ч", "th": "с", "zh": "ж",
	"ph": "ф", "wh": "в", "ck": "к", "ng": "нг",
	"qu": "кв", "ew": "ю",
}

// Single letter translit
var letters = map[rune]string{
	'a': "а", 'b': "б", 'c': "си", 'd': "д",
	'e': "е", 'f': "ф", 'g': "г", 'h': "х",
	'i': "и", 'j': "дж", 'k': "к", 'l': "л",
	'm': "м", 'n': "н", 'o': "о", 'p': "п",
	'q': "к", 'r': "р", 's': "с", 't': "т",
	'u': "у", 'v': "в", 'w': "в", 'x': "кс",
	'y': "й", 'z': "з",
}

var engWordRegex = regexp.MustCompile(`[a-zA-Z]+`)

func transliterate(input string) string {
	return engWordRegex.ReplaceAllStringFunc(input, func(word string) string {
		return transliterateWord(word)
	})
}

func transliterateWord(word string) string {
	word = strings.ToLower(word)
	var sb strings.Builder
	i := 0
	for i < len(word) {
		if i+1 < len(word) {
			pair := word[i : i+2]
			if val, ok := digraphs[pair]; ok {
				sb.WriteString(val)
				i += 2
				continue
			}
		}
		ch := rune(word[i])
		if val, ok := letters[ch]; ok {
			sb.WriteString(val)
		} else {
			sb.WriteRune(ch)
		}
		i++
	}
	return sb.String()
}
