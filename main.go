package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/CanalTP/gormungandr/kraken"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func directPath(k kraken.Kraken, coords []string, quiet bool) {
	rand.Seed(time.Now().UnixNano())
	choosenFrom := coords[rand.Intn(len(coords))]
	choosenTo := coords[rand.Intn(len(coords))]
	modes := []string{"walking", "car", "bike"}
	choosenMode := modes[rand.Intn(len(modes))]

	dp := kraken.DirectPathBuilder{
		From:   choosenFrom,
		To:     choosenTo,
		Mode:   choosenMode,
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

func matrix(k kraken.Kraken, coords []string, maxDuration int32, quiet bool) {
	rand.Seed(time.Now().UnixNano())
	choosenFrom := []string{coords[rand.Intn(len(coords))]}
	modes := []string{"walking", "car", "bike"}
	choosenMode := modes[rand.Intn(len(modes))]

	matrix := kraken.StreetNetworkMatrixBuilder{
		From:        choosenFrom,
		To:          coords,
		Mode:        choosenMode,
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
	var coordFile string

	var cmdDP = &cobra.Command{
		Use:   "directpath --coords <coordsFile>",
		Short: "compute a direct path",
		Long:  `compute a direct path from a random coord to an other random coord`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			k := kraken.NewKrakenZMQ("test", target, timeout)
			coords := args
			var err error
			if coordFile != "" {
				coords, err = LoadCoordFromFile(coordFile)
				if err != nil {
					logrus.Fatal(err)
				}
			}
			f := func() {
				directPath(k, coords, quiet)
			}
			if bench.Seconds() < 1 {
				f()
			} else {
				benchmark(bench, goroutines, f)
			}
		},
	}

	var cmdMatrix = &cobra.Command{
		Use:   "matrix --coords <coordsFile> ...",
		Short: "compute a matrix",
		Long:  `compute a matrix from a random coord to all the others`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			k := kraken.NewKrakenZMQ("test", target, timeout)
			coords := args
			var err error
			if coordFile != "" {
				coords, err = LoadCoordFromFile(coordFile)
				if err != nil {
					logrus.Fatal(err)
				}
			}
			f := func() {
				matrix(k, coords, maxDuration, quiet)
			}
			if bench.Seconds() < 1 {
				f()
			} else {
				benchmark(bench, goroutines, f)
			}
		},
	}
	cmdMatrix.Flags().Int32Var(&maxDuration, "max-duration", 30*60, "max duration to explore")

	rootCmd := &cobra.Command{}
	rootCmd.PersistentFlags().StringVar(&coordFile, "coords", "", "file to read coordinates")
	rootCmd.PersistentFlags().StringVarP(&target, "target", "t", "tcp://127.0.0.1:6000", "kraken to target")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "remove normal output")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "d", 10*time.Second, "kraken timeout")
	rootCmd.PersistentFlags().DurationVar(&bench, "bench", 0*time.Second, "run the benchmark for the specified duration")
	rootCmd.PersistentFlags().IntVarP(&goroutines, "concurrency", "c", 1, "number of goroutine to launch in bench modes")
	rootCmd.AddCommand(cmdDP)
	rootCmd.AddCommand(cmdMatrix)
	_ = rootCmd.Execute()
}
