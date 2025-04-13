// Copyright (c) 2020 Polyverse Corporation

package main

//TODO: CLEAN UP, REFACTOR

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var yakFile string
var lexFile string

const y_check = "zend_language_parser.y"
const l_check = "zend_language_scanner.l"
const zend_dir = "/Zend/"
const source_env_var = "PHP_SRC_PATH"

type QuotedStringOperator = func(string) string

func init() {
	dictFlag := flag.String("dict", "", "String: Prexisting scrambled JSON dictionary.")
	charFlag := flag.Bool("chars", true, "Boolean: Scramble Character Tokens")
	checkEnvs()
	flag.Parse()
	dictFile := *dictFlag
	charScram := *charFlag
	KeywordsRegex.Longest()
	if dictFile != "" {
		InitEEWords(dictFile)
	} else if charScram {
		InitChar()
	}
	checkTokens(lexFile)
}

func main() {
	scanLines(lexFile, []byte("<ST_IN_SCRIPTING>"), true)
	fmt.Println("Mapping Built. \nLex Scrambled.")
	Buffer.Reset()
	scanLines(yakFile, []byte("%token T_"), false)
	fmt.Println("Yak Scrambled.")
	SerializeMap()
	fmt.Println("Map Serialized.")
}

func checkTokens(lexFile string) {
	file, err := ioutil.ReadFile(lexFile)
	Check(err)

	const tokens = "TOKENS [;:,.|^&+-/*=%!~$<>?@"
	const spec_case = "[[(){}\"`"

	lines := strings.Split(string(file), "\n")
	for i, line := range lines {
		if strings.Contains(line, tokens) {
			outLine := tokens
			for key, val := range SpecialChar {
				outLine = strings.Replace(outLine, val, key, 1)
			}
			lines[i] = outLine + "]"
		} else if strings.Contains(line, spec_case) {
			outLine := spec_case
			for key, val := range SpecialChar {
				outLine = strings.Replace(outLine, val, key, 1)
			}
			lines[i] = "<ST_VAR_OFFSET>{TOKENS}|" + outLine + "] {"
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(lexFile, []byte(output), 0644)
	if err != nil {
		log.Fatalln(err)
	}

}

func scanLines(fileIn string, flag []byte, scanNextLine bool) {
	file, err := os.Open(fileIn)
	Check(err)

	fileScanner := bufio.NewScanner(file)

	for fileScanner.Scan() {
		line := fileScanner.Bytes()

		if bytes.HasPrefix(line, flag) && KeywordsRegex.Match(line) {
			line = getWords(line, false)

			// occasionally the next line may also contain the same keyword (in the rule). If so, process it.
			if scanNextLine && fileScanner.Scan() {
				nextline := fileScanner.Bytes()
				nextline = getWords(nextline, true)
				// append nextline to line
				line = append(append(line, []byte("\n")...), nextline...)
			}
		} else if bytes.HasPrefix(line, flag) && CharStrRegex.Match(line) {
			line = getCharStr(line)
		}

		if hasChar(line) {
			line = getChar(line)
		}

		WriteLineToBuff(line)
	}

	WriteFile(fileIn)
	err = file.Close()
	Check(err)
}

func getWords(s []byte, mustBeQuoted bool) []byte {
	if mustBeQuoted {
		return inMatchingQuotes(s, substituteWordsInString)
	} else {
		return []byte(substituteWordsInString(string(s)))
	}
}

func substituteWordsInString(line string) string {
	matchedRegexStart := KeywordsRegex.FindString(line)
	matchedRegex := KeywordsRegex.FindString(line)

	for matchedRegex != "" {
		index := KeywordsRegex.FindStringIndex(line)
		suffix := string(line[index[1]-1])
		prefix := string(line[index[0]])

		matchedRegex = strings.TrimSuffix(strings.TrimPrefix(matchedRegex, prefix), suffix)
		key := strings.TrimPrefix(matchedRegex, "\"")

		if _, ok := GetScrambled(key); ok || PreMadeDict {
			key, _ = GetScrambled(key)
		} else {
			AddToEEWords(strings.ToLower(key))
			key, _ = GetScrambled(strings.ToLower(key))
		}

		line = strings.Replace(line, strings.TrimPrefix(matchedRegex, "\""), key, 1)
		matchedRegex = KeywordsRegex.FindString(line)

		if matchedRegex == matchedRegexStart {
			fmt.Println(matchedRegex + ": Not added to dictionary.")
			return line
		}
	}

	return line
}

func hasChar(line []byte) bool {
	var stringifiedline = string(line)

	for _, charMatch := range CharMatches {
		if strings.Contains(stringifiedline, charMatch) {
			return true
		}
	}

	return false
}

func getChar(line []byte) []byte {
	GetScrambledWrapper := func(l string) string {
		r, _ := GetScrambled(l)
		return r
	}
	return inMatchingQuotes(line, GetScrambledWrapper)
}

func inMatchingQuotes(line []byte, operator QuotedStringOperator) []byte {
	replace := bytes.NewBufferString("")

	var doubleQuote = byte('"')
	var singleQuote = byte('\'')

	var inDoubleQuote = false
	var inSingleQuote = false

	cache := bytes.NewBufferString("")

	for i := 0; i < len(line); i++ {
		if inSingleQuote && line[i] == singleQuote {
			inSingleQuote = false
			var substitution = operator(cache.String())
			replace.WriteString(substitution)
			replace.WriteByte(line[i])
		} else if inDoubleQuote && line[i] == doubleQuote {
			inDoubleQuote = false
			var substitution = operator(cache.String())
			replace.WriteString(substitution)
			replace.WriteByte(line[i])
		} else if inSingleQuote || inDoubleQuote {
			cache.WriteByte(line[i])
		} else if line[i] == singleQuote {
			inSingleQuote = true
			replace.WriteByte(line[i])
			cache = bytes.NewBufferString("")
		} else if line[i] == doubleQuote {
			inDoubleQuote = true
			replace.WriteByte(line[i])
			cache = bytes.NewBufferString("")
		} else {
			replace.WriteByte(line[i])
		}
	}

	return replace.Bytes()
}

func getCharStr(line []byte) []byte {
	return CharStrRegex.ReplaceAllFunc(line, replaceFunction)
}

func replaceFunction(src []byte) []byte {
	var replace string
	for i := 0; i < len(src); i++ {
		char, _ := GetScrambled(string(src[i]))
		replace += char
	}
	return []byte(replace)
}

func checkEnvs() {
	var phpSrc = os.Getenv(source_env_var)

	if phpSrc == "" {
		l := log.New(os.Stderr, "", 0)
		l.Println("No PHP Source Path Found. Continuing in current directory.")
		yakFile = y_check
		lexFile = l_check
		return
	}
	yakFile = phpSrc + zend_dir + y_check
	lexFile = phpSrc + zend_dir + l_check
}
