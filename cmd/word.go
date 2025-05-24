package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sync"

	"github.com/spf13/cobra"
)

type wordStats struct {
	Name  string
	Words int
}

type resultWord struct {
	index int
	stats wordStats
	err   error
}

var wordCmd = &cobra.Command{
	Use:   "word [файлы...]",
	Short: "Подсчет слов в файлах",
	Long: `Команда word подсчитывает количество слов 
в указанных файлах или вводе из стандартного потока`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			stats, err := countWordsFromReader(os.Stdin, "stdin")
			if err != nil {
				cmd.PrintErrln("Ошибка:", err)
				os.Exit(1)
			}

			printWordStats(cmd, stats)
			return
		}

		var wg sync.WaitGroup
		ch := make(chan resultWord, len(args))
		var hasErrors bool
		var totalWords int

		for i, filename := range args {
			wg.Add(1)
			go func(i int, filename string) {
				defer wg.Done()
				stats, err := countWordsInFile(filename)
				ch <- resultWord{index: i, stats: stats, err: err}
			}(i, filename)
		}

		wg.Wait()
		close(ch)

		results := make([]resultWord, len(args))
		for res := range ch {
			results[res.index] = res
		}

		for _, res := range results {
			if res.err != nil {
				cmd.PrintErrf("Ошибка обработки файла %s: %v\n", res.stats.Name, res.err)
				hasErrors = true
			} else {
				printWordStats(cmd, res.stats)
				totalWords += res.stats.Words
			}
		}

		if len(args) > 1 {
			cmd.Printf("\nВсего слов: %d\n", totalWords)
		}

		if hasErrors {
			os.Exit(1)
		}
	},
}

func countWordsInFile(filename string) (wordStats, error) {
	file, err := os.Open(filename)
	if err != nil {
		return wordStats{Name: filename}, fmt.Errorf("Ошибка открытия файла: %w", err)
	}
	defer file.Close()

	return countWordsFromReader(file, filename)
}

func countWordsFromReader(r io.Reader, name string) (wordStats, error) {
	scanner := bufio.NewScanner(r)
	stats := wordStats{Name: name}
	wordRegex := regexp.MustCompile(`[\p{L}\p{N}_'-]+`)

	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		words := wordRegex.FindAllString(line, -1)
		stats.Words += len(words)
	}
	if err := scanner.Err(); err != nil {
		return wordStats{}, fmt.Errorf("Ошибка чтения: %w", err)
	}

	return stats, nil
}

func printWordStats(cmd *cobra.Command, stats wordStats) {
	cmd.Printf("Файл: %s\n", stats.Name)
	cmd.Printf("Слов: %d\n", stats.Words)
	cmd.Println("──────────────────────────────")
}

func init() {
	rootCmd.AddCommand(wordCmd)
}
