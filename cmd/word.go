package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/spf13/cobra"
)

type wordStats struct {
	Name  string
	Words int
}

var wordCmd = &cobra.Command{
	Use:   "word [файлы...]",
	Short: "Подсчет слов в файлах",
	Long:  `Команда word подсчитывает количество слов 
в указанных файлах или вводе из стандартного потока`,
	Args: cobra.ArbitraryArgs,
	Run:  func(cmd *cobra.Command, args []string) {
		var totalWords int
		var hasErrors bool

		if len(args) == 0 {
			stats, err := countWordsFromReader(os.Stdin, "stdin")
			if err != nil {
				cmd.PrintErrln("Ошибка:", err)
				os.Exit(1)
			}

			printWordStats(cmd, stats)
			return
		}

		for _, filename := range args {
			stats, err := countWordsInFile(filename)
			if err != nil {
				cmd.PrintErrf("Ошибка обработки файла %s: %v\n", filename, err)
				hasErrors = true
				continue
			}

			totalWords += stats.Words
			printWordStats(cmd, stats)
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
		return wordStats{}, fmt.Errorf("Ошибка открытия файла: %w", err)
	}
	defer file.Close()

	return countWordsFromReader(file, filename)
}

func countWordsFromReader(r io.Reader, name string) (wordStats, error) {
	scanner   := bufio.NewScanner(r)
	stats     := wordStats{Name: name}
	wordRegex := regexp.MustCompile(`[\p{L}\p{N}_'-]+`)

	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	
	for scanner.Scan() {
		line  := scanner.Text()
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
