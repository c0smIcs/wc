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
	"regexp"
	"unicode"

	"fmt"
	"io"
	"os"

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

var (
	countLetters bool
	countWords   bool
	countBytes   bool
)

const (
	configLetters  = "letters"
	configWords    = "words"
	configBytes    = "bytes"
	configFormat   = "format"
	configFileName = ".wcrc"
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
	Run:  func(cmd *cobra.Command, args []string) {
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

		for _, filename := range files {
			stats, err := calculateFileStats(filename)
			if err != nil {
				cmd.PrintErrln("Ошибка:", err)
				continue
			}

			totalStats.Name    += "TOTAL"
			totalStats.Letters += stats.Letters
			totalStats.Lines   += stats.Lines
			totalStats.Words   += stats.Words
			totalStats.Bytes   += stats.Bytes

			printStats(cmd, stats)
		}

		if len(files) > 1 {
			cmd.Println("\nОбщая статистика:")
			printStats(cmd, totalStats)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().BoolP(configLetters,  "l",  false,  "Подсчет букв")
	rootCmd.Flags().BoolP(configWords,    "w",  false,  "Подсчет слов")
	rootCmd.Flags().BoolP(configBytes,    "b",  false,  "Подсчет байтов")
	rootCmd.Flags().StringP(configFormat, "f",  "text", "Формат вывода (text|json|csv)")

	viper.BindPFlag(configLetters, rootCmd.Flags().Lookup(configLetters))
	viper.BindPFlag(configWords,   rootCmd.Flags().Lookup(configWords))
	viper.BindPFlag(configBytes,   rootCmd.Flags().Lookup(configBytes))
	viper.BindPFlag(configFormat,  rootCmd.Flags().Lookup(configFormat))
}

func initConfig() {
	viper.SetEnvPrefix("WC")
	viper.AutomaticEnv()

	viper.SetConfigName(configFileName)
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Println("Ошибка конфигурации:", err)
			return
		}
	}
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
	stats     := fileStats{Name: name}
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
	
	cmd.Printf("Lines:   %d\n",   stats.Lines)
	cmd.Println("──────────────────────────────")
}

func anyFlagsSet() bool {
	return countLetters || countWords || countBytes
}

// фильтр для отделения имен файлов от флагов
func filterArgs(args []string) []string {
	var files []string
	for _, arg := range args {
		if arg[0] != '-' {
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
