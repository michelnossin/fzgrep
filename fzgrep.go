package main

import (

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"fmt"
	"github.com/c-bata/go-prompt"
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"log"
)

var root_path, err, query, path string
var suggestions []prompt.Suggest
var wg sync.WaitGroup
var waiting_for_wg = false
var max_suggestions = 10
var minimum_query_size_before_grep = 3
var show_surrounding_characters = 30

func stripCtlAndExtFromUnicode(str string) string {
	isOk := func(r rune) bool {
		return r < 32 || r >= 127
	}
	t := transform.Chain(norm.NFKD, transform.RemoveFunc(isOk))
	str, _, _ = transform.String(t, str)
	return str
}


func queryInFile(wg *sync.WaitGroup, path string) {
	defer wg.Done()

	file, err := os.Open(path)
	defer file.Close()

	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	for i := 1; scanner.Scan(); i++ {

		text_to_scan := stripCtlAndExtFromUnicode(scanner.Text())
		if strings.Contains(text_to_scan, query) && (len(suggestions) <= max_suggestions) && !waiting_for_wg {
			start_substring := strings.Index(text_to_scan, query)
			end_substring := start_substring + len(query)
                        start_substring = start_substring - show_surrounding_characters
			if start_substring < 0 {
				start_substring = 0
			}
			end_substring = end_substring + show_surrounding_characters
			if end_substring > len(text_to_scan) - 1 {
				end_substring = len(text_to_scan) - 1
			}
			grep_suggestions := text_to_scan[start_substring:end_substring]
			item := prompt.Suggest{Text: path, Description: grep_suggestions}
			suggestions = append(suggestions, item)
		}
		if (len(suggestions) > max_suggestions) || waiting_for_wg {
			break
		}
	}
}


func completer(d prompt.Document) []prompt.Suggest {

	waiting_for_wg = true
	wg.Wait()
	waiting_for_wg = false

	suggestions = []prompt.Suggest{}
	query = d.GetWordBeforeCursor()
	if len(query) < minimum_query_size_before_grep {
		return nil
	}

	filepath.Walk(root_path, func(path string, file os.FileInfo, err error) error {
		if !file.IsDir() {
			wg.Add(1)
			go queryInFile(&wg, path)
		}
		return nil
	})

	if len(suggestions) == 0 {
		suggestions = []prompt.Suggest{
			{Text: "Not found", Description: "Not found"},
		}
	}

	return suggestions
}

func main() {
	current_dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	root_path = current_dir
	fmt.Println("Please enter query pattern:")
	selected_file_grep := prompt.Input("> ", completer)
	fmt.Println("You selected " + selected_file_grep)
}
