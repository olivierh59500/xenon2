// Command import-assets verifies and decodes local original resources.
package main

import (
	"flag"
	"fmt"
	"os"

	"xenon2/internal/assetimport"
)

func main() {
	adf := flag.String("adf", "", "user-supplied supported Xenon II AmigaDOS ADF")
	output := flag.String("output", "assets/original", "excluded packed asset directory")
	analysis := flag.String("analysis", ".local/imported", "excluded decoded analysis directory")
	dryRun := flag.Bool("dry-run", false, "validate all resources without writing")
	verify := flag.Bool("verify", false, "validate previously imported local resources")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("use -adf to specify the original disk"))
	}
	if *verify {
		if *adf != "" || *dryRun {
			fail(fmt.Errorf("-verify cannot be combined with -adf or -dry-run"))
		}
		if err := assetimport.Validate(*output, *analysis); err != nil {
			fail(err)
		}
		fmt.Printf("Verified %d asset containers and the offline executable image.\n", len(assetimport.Levels))
		return
	}
	count, err := assetimport.Import(assetimport.Config{ADF: *adf, Output: *output, Analysis: *analysis, DryRun: *dryRun})
	if err != nil {
		fail(err)
	}
	if *dryRun {
		fmt.Printf("Validated %d resources; no files were written.\n", count)
		return
	}
	fmt.Printf("Imported %d resources. Original program data remains confined to %s.\n", count, *analysis)
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
