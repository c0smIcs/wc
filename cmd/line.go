package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

type lineStats struct {
	Name  string
	Lines int
}

type resultLine struct {
	index int
	stats lineStats
	err   error
}

var lineCmd = &cobra.Command{
	Use:   "line [файлы...]",
	Short: "Подсчет строк в файлах",
	Long: `Команда line подсчитывает количество строк 
в указанных файлах или вводе из стандартного потока`,
	Example: `
wc line file.txt         # Анализ одного файла
wc line *.txt            # Анализ нескольких файлов
cat file.txt | wc line   # Анализ из stdin`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			stats, err := countLinesFromReader(os.Stdin, "stdin")
			if err != nil {
				cmd.PrintErrln("Ошибка:", err)
				os.Exit(1)
			}
			printLineStats(cmd, stats)
			return
		}

		var wg sync.WaitGroup
		ch := make(chan resultLine, len(args))
		var hasErrors bool
		var totalLines int

		for i, filename := range args {
			wg.Add(1)
			go func(i int, filename string) {
				defer wg.Done()
				stats, err := countLinesInFile(filename)
				ch <- resultLine{index: i, stats: stats, err: err}
			}(i, filename)
		}

		wg.Wait()
		close(ch)

		results := make([]resultLine, len(args)) 
		for res := range ch {
			results[res.index] = res
		}

		for _, res := range results {
			if res.err != nil {
				cmd.PrintErrf("Ошибка обработки файла %s: %v\n", args[res.index], res.err)
				hasErrors = true
			} else {
				printLineStats(cmd, res.stats)
				totalLines += res.stats.Lines
			}
		}

		if len(args) > 1 {
			cmd.Printf("\nВсего строк: %d\n", totalLines)
		}

		if hasErrors {
			os.Exit(1)
		}
	},
}

func countLinesInFile(filename string) (lineStats, error) {
	file, err := os.Open(filename)
	if err != nil {
		return lineStats{}, fmt.Errorf("Ошибка открытия файла: %w", err)
	}
	defer file.Close()

	return countLinesFromReader(file, filename)
}

func countLinesFromReader(r io.Reader, name string) (lineStats, error) {
	scanner := bufio.NewScanner(r)
	stats := lineStats{Name: name}

	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		stats.Lines++
	}

	if err := scanner.Err(); err != nil {
		return lineStats{}, fmt.Errorf("Ошибка чтения: %w", err)
	}

	return stats, nil
}

func printLineStats(cmd *cobra.Command, stats lineStats) {
	cmd.Printf("Файл: %s\n", stats.Name)
	cmd.Printf("Строк: %d\n", stats.Lines)
	cmd.Println("──────────────────────────────")
}

func init() {
	rootCmd.AddCommand(lineCmd)
}
