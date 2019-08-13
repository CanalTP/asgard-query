package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CanalTP/gormungandr/kraken"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func directPath(k kraken.Kraken, from, to string, quiet bool) {
	dp := kraken.DirectPathBuilder{
		From:   from,
		To:     to,
		Kraken: k,
	}
	r, err := dp.Get()
	if err != nil {
		logrus.Error(err)
	}
	m := jsonpb.Marshaler{Indent: "  "}
	j, err := m.MarshalToString(r)
	if err != nil {
		logrus.Error(err)
	}
	if !quiet {
		fmt.Print(j)
	} else {
		fmt.Print(".")
	}
}

func matrix(k kraken.Kraken, from, to []string, maxDuration int32, quiet bool) {
	matrix := kraken.StreetNetworkMatrixBuilder{
		From:        from,
		To:          to,
		Kraken:      k,
		MaxDuration: maxDuration,
	}
	r, err := matrix.Get()
	if err != nil {
		logrus.Error(err)
	}
	m := jsonpb.Marshaler{Indent: "  "}
	j, err := m.MarshalToString(r)
	if err != nil {
		logrus.Error(err)
	}
	if !quiet {
		fmt.Print(j)
	} else {
		fmt.Print(".")
	}
}

func LoadCoordFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	result := make([]string, 0)
	if err != nil {
		return nil, errors.Wrap(err, "Open failed")
	}
	defer func() { file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if t != "" {
			result = append(result, t)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func benchmark(duration time.Duration, concurrency int, f func()) {
	loop := func() {
		for {
			f()
		}
	}
	for i := 0; i < concurrency; i++ {
		go loop()
	}
	<-time.After(duration)
}

func main() {
	var target string
	var quiet bool
	var timeout time.Duration
	var bench time.Duration
	var goroutines int
	var maxDuration int32

	var cmdDP = &cobra.Command{
		Use:   "directpath <from> <to>",
		Short: "compute a direct path",
		Long:  `compute a direct path from "from" to "to"`,
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			k := kraken.NewKrakenZMQ("test", target, timeout)
			f := func() {
				directPath(k, args[0], args[1], quiet)
			}
			if bench.Seconds() < 1 {
				f()
			} else {
				benchmark(bench, goroutines, f)
			}
		},
	}

	var fromFile, toFile string
	var cmdMatrix = &cobra.Command{
		Use:   "matrix <coord> <coord> <coord> ...",
		Short: "compute a direct path",
		Long:  `compute a direct path from "from" to "to"`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			k := kraken.NewKrakenZMQ("test", target, timeout)
			from := args
			to := args
			var err error
			if fromFile != "" {
				from, err = LoadCoordFromFile(fromFile)
				if err != nil {
					logrus.Fatal(err)
				}
			}
			if toFile != "" {
				to, err = LoadCoordFromFile(toFile)
				if err != nil {
					logrus.Fatal(err)
				}
			}
			f := func() {
				matrix(k, from, to, maxDuration, quiet)
			}
			if bench.Seconds() < 1 {
				f()
			} else {
				benchmark(bench, goroutines, f)
			}
		},
	}
	cmdMatrix.Flags().StringVar(&fromFile, "from", "", "file to read from coordinates")
	cmdMatrix.Flags().StringVar(&toFile, "to", "", "file to read to coordinates")
	cmdMatrix.Flags().Int32Var(&maxDuration, "max-duration", 30*60, "max duration to explore")

	rootCmd := &cobra.Command{}
	rootCmd.PersistentFlags().StringVarP(&target, "target", "t", "tcp://127.0.0.1:6000", "kraken to target")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "remove normal output")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "d", 10*time.Second, "kraken timeout")
	rootCmd.PersistentFlags().DurationVar(&bench, "bench", 0*time.Second, "run the benchmark for the specified duration")
	rootCmd.PersistentFlags().IntVarP(&goroutines, "concurrency", "c", 1, "number of goroutine to launch in bench modes")
	rootCmd.AddCommand(cmdDP)
	rootCmd.AddCommand(cmdMatrix)
	_ = rootCmd.Execute()
}
