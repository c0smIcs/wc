package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"unicode"

	"github.com/spf13/cobra"
)

type charStats struct {
	Name    string
	Letters int
}

type resultChar struct {
	index int
	stats charStats
	err   error
}

var charCmd = &cobra.Command{
	Use:   "char [файлы...]",
	Short: "Подсчет буквенных символов в файлах",
	Long: `Команда char подсчитывает количество буквенных символов 
в указанных файлах или вводе из стандартного потока`,
	Example: `
wc char file.txt         # Анализ одного файла
wc char *.txt            # Анализ нескольких файлов
cat file.txt | wc char   # Анализ из stdin`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			stats, err := countLettersFromReader(os.Stdin, "stdin")
			if err != nil {
				cmd.PrintErrln("Ошибка", err)
				os.Exit(1)
			}
			printCharStats(cmd, stats)
			return
		}

		var wg sync.WaitGroup
		ch := make(chan resultChar, len(args))
		var hasErrors bool
		var totalLetters int

		for i, filename := range args {
			wg.Add(1)
			go func(i int, filename string) {
				defer wg.Done()
				stats, err := countLettersInFile(filename)
				ch <- resultChar{index: i, stats: stats, err: err}
			}(i, filename)
		}

		go func() {
			wg.Wait()
			close(ch)
		}()

		results := make([]resultChar, len(args))
		for res := range ch {
			results[res.index] = res
		}

		for _, res := range results {
			if res.err != nil {
				cmd.PrintErrf("Ошибка обработки файла %s: %v\n", args[res.index], res.err)
				hasErrors = true
			} else {
				printCharStats(cmd, res.stats)
				totalLetters += res.stats.Letters
			}
		}
		if len(args) > 1 {
			cmd.Printf("\nВсего букв: %d\n", totalLetters)
		}

		if hasErrors {
			os.Exit(1)
		}
	},
}

func countLettersInFile(filename string) (charStats, error) {
	file, err := os.Open(filename)
	if err != nil {
		return charStats{}, fmt.Errorf("Ошибка открытия файла: %w", err)
	}
	defer file.Close()

	return countLettersFromReader(file, filename)
}

func countLettersFromReader(r io.Reader, name string) (charStats, error) {
	scanner := bufio.NewScanner(r)
	stats := charStats{Name: name}

	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	scanner.Split(bufio.ScanRunes)

	for scanner.Scan() {
		char := rune(scanner.Bytes()[0])
		if unicode.IsLetter(char) {
			stats.Letters++
		}
	}

	if err := scanner.Err(); err != nil {
		return charStats{}, fmt.Errorf("Ошибка чтения: %w", err)
	}

	return stats, nil
}

func printCharStats(cmd *cobra.Command, stats charStats) {
	cmd.Printf("Файл: %s\n", stats.Name)
	cmd.Printf("Букв: %d\n", stats.Letters)
	cmd.Println("──────────────────────────────")
}

func init() {
	rootCmd.AddCommand(charCmd)
}
