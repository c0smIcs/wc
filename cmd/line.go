package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

type lineStats struct {
	Name  string
	Lines int
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
		var totalLines int
		var hasErrors bool

		if len(args) == 0 {
			stats, err := countLinesFromReader(os.Stdin, "stdin")
			if err != nil {
				cmd.PrintErrln("Ошибка:", err)
				os.Exit(1)
			}
			printLineStats(cmd, stats)
			return
		}

		for _, filename := range args {
			stats, err := countLinesInFile(filename)
			if err != nil {
				cmd.PrintErrf("Ошибка обработки файла %s: %v\n", filename, err)
				hasErrors = true
				continue 
			}

			totalLines += stats.Lines
			printLineStats(cmd, stats)
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
	stats   := lineStats{Name: name}

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
	cmd.Printf("Файл: %s\n",  stats.Name)
	cmd.Printf("Строк: %d\n", stats.Lines)
	cmd.Println("──────────────────────────────")
}

func init() {
	rootCmd.AddCommand(lineCmd)
}
