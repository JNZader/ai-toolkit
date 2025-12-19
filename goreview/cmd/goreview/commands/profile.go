package commands

import (
	"os"
	"runtime/pprof"

	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:    "profile",
	Short:  "Run with profiling enabled",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// CPU profiling
		cpuFile, err := os.Create("cpu.prof")
		if err != nil {
			return err
		}
		defer cpuFile.Close()

		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			return err
		}
		defer pprof.StopCPUProfile()

		// Run review (pass arguments manually)
		if err := runReview(cmd, args); err != nil {
			return err
		}

		// Memory profiling
		memFile, err := os.Create("mem.prof")
		if err != nil {
			return err
		}
		defer memFile.Close()

		return pprof.WriteHeapProfile(memFile)
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
}
