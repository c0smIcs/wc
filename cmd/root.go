/*
Какие улучшения можно внести:

 1. Конфигурируемость:
    Добавить флаг для выбора типа символов (только ASCII или все Unicode буквы)
    Добавить флаг для игнорирования регистра

 2. Производительность:
    Добавить параллельную обработку файлов

 3. Вывод:
    Добавить форматирование таблицей
    Поддержку JSON/CSV вывода через флаги

 4. Тестирование:
    Добавить unit-тесты для функции подсчета
    Проверку на edge-cases (пустые файлы, бинарные файлы)

 5. Документация:
    Добавить примеры использования в Long описание
    Указать в документации, что считается буквой

___________________________________________________________________

2. Поддержка форматов вывода (JSON/CSV)
Упростит интеграцию с другими инструментами и скриптами

3. Игнорирование регистра букв
Для корректного подсчета в языках с регистровыми буквами.

4. Конфигурация через Viper
Хранение настроек по умолчанию (например, формата вывода)
*/
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type fileStats struct {
	Name    string
	Letters int
	Lines   int
	Words   int
	Bytes   int
}

type resultRoot struct {
	stats fileStats
	err   error
}

var (
	countLetters bool
	countWords   bool
	countBytes   bool
	outputFormat string
)

const (
	configLetters  = "letters"
	configWords    = "words"
	configBytes    = "bytes"
	configFormat   = "format"
	configFileName = "configYaml"
)

var rootCmd = &cobra.Command{
	Use:   "wc [файлы...]",
	Short: "Утилита для подсчета символов, строк и слов в файле",
	Long: `WC - это аналог UNIX-утилиты для анализа текста.
Данная утилита может подсчитывать:
- Количество строк  (всегда отображается)
- Количество букв   (-l)
- Количество слов   (-w)
- Количество байтов (-b)

Можно также передать файлы или читать из stdin (например, через pipe).`,
	Example: `
wc file.txt -l -w           # Анализ одного файла
wc file1.txt file2.txt -c   # Анализ нескольких файлов
cat file.txt | wc -l -w     # Анализ из stdin`,

	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		files := filterArgs(args)

		if len(files) == 0 {
			stats, err := calculateStats(os.Stdin, "stdin")
			if err != nil {
				cmd.PrintErrf("Ошибка чтения stdin: %v\n", err)
				return
			}
			printStats(cmd, stats)
			return
		}

		var totalStats fileStats
		totalStats.Name = "TOTAL"

		results := make([]resultRoot, len(files))
		var wg sync.WaitGroup
		wg.Add(len(files))

		for i, filename := range files {
			go func(i int, filename string) {
				defer wg.Done()
				stats, err := calculateFileStats(filename)
				
				results[i] = resultRoot{stats: stats, err: err}
			}(i, filename)
		}

		wg.Wait()

		for _,  res := range results {
			if res.err != nil {
				cmd.PrintErrln("Ошибка: ", res.err)
			} else {
				printStats(cmd, res.stats)
				totalStats.Letters += res.stats.Letters
				totalStats.Lines += res.stats.Lines
				totalStats.Words += res.stats.Words
				totalStats.Bytes += res.stats.Bytes
			}
		}

		if len(files) > 1 {
			cmd.Println("\nОбщая статистика:")
			printStats(cmd, totalStats)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().BoolP(configLetters, "l", false, "Подсчет букв")
	rootCmd.Flags().BoolP(configWords, "w", false, "Подсчет слов")
	rootCmd.Flags().BoolP(configBytes, "b", false, "Подсчет байтов")
	rootCmd.Flags().StringP(configFormat, "f", "text", "Формат вывода (text|json|csv)")

	viper.BindPFlag(configLetters, rootCmd.Flags().Lookup(configLetters))
	viper.BindPFlag(configWords, rootCmd.Flags().Lookup(configWords))
	viper.BindPFlag(configBytes, rootCmd.Flags().Lookup(configBytes))
	viper.BindPFlag(configFormat, rootCmd.Flags().Lookup(configFormat))
}

func initConfig() {
	viper.SetEnvPrefix("WC")
	viper.AutomaticEnv()

	viper.SetConfigName(configFileName)
	viper.AddConfigPath("PATH")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Println("Ошибка конфигурации:", err)
			return
		}
	}

	countLetters = viper.GetBool(configLetters)
	countWords = viper.GetBool(configWords)
	countBytes = viper.GetBool(configBytes)
	outputFormat = viper.GetString(configFormat)
}

func calculateFileStats(filename string) (fileStats, error) {
	file, err := os.Open(filename)
	if err != nil {
		return fileStats{}, fmt.Errorf("не удалось открыть файл %s: %v", filename, err)
	}
	defer file.Close()

	return calculateStats(file, filename)
}

func calculateStats(r io.Reader, name string) (fileStats, error) {
	bufReader := bufio.NewReader(r)
	stats := fileStats{Name: name}
	wordRegex := regexp.MustCompile(`\p{L}+`)

	for {
		line, err := bufReader.ReadString('\n')
		stats.Bytes += len(line)

		if err != nil && err != io.EOF {
			return fileStats{}, fmt.Errorf("Ошибка чтения %s: %v", name, err)
		}

		if line != "" {
			stats.Lines++

			for _, char := range line {
				if unicode.IsLetter(char) {
					stats.Letters++
				}
			}

			words := wordRegex.FindAllString(line, -1)
			stats.Words += len(words)
		}

		if err == io.EOF {
			break
		}
	}

	return stats, nil
}

func printStats(cmd *cobra.Command, stats fileStats) {
	switch outputFormat {

	case "text":
		cmd.Printf("Файл:    %s\n", stats.Name)
		if countLetters || !anyFlagsSet() {
			cmd.Printf("Letters: %d\n", stats.Letters)
		}
		if countWords || !anyFlagsSet() {
			cmd.Printf("Words:   %d\n", stats.Words)
		}
		if countBytes || !anyFlagsSet() {
			cmd.Printf("Bytes:   %d\n", stats.Bytes)
		}
		cmd.Printf("Lines:   %d\n", stats.Lines)
		cmd.Println("──────────────────────────────")

	case "json":
		cmd.Printf(`{
	"name":   "%s",
	"letters": %d,
	"words":   %d,
	"bytes":   %d,
	"lines":   %d,
}`+"\n",
			stats.Name, stats.Letters, stats.Words, stats.Bytes, stats.Lines)

	case "csv":
		cmd.Printf("name,letters,words,bytes,lines\n")
		cmd.Printf("%s,%d,%d,%d,%d\n", stats.Name, stats.Letters, stats.Words, stats.Bytes, stats.Lines)

	default:
		cmd.Println("Неизвестный формат вывода.")
	}
}

func anyFlagsSet() bool {
	return countLetters || countWords || countBytes
}

func filterArgs(args []string) []string {
	var files []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			files = append(files, arg)
		}
	}
	return files
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
