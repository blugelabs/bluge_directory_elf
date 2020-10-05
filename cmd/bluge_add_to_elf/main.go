//  Copyright (c) 2020 The Bluge Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/blugelabs/bluge/index"
	"github.com/blugelabs/bluge_directory_elf"
)

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatalf("must specify path to elf executable")
	} else if flag.NArg() < 2 {
		log.Fatalf("must specify new index name")
	} else if flag.NArg() < 3 {
		log.Fatalf("must specify path to bluge index")
	}

	var i int
	var inputElfPath = flag.Arg(0)
	var outputElfPath string
	err := filepath.Walk(flag.Arg(2), func(path string, info os.FileInfo, err error) error {
		if err == nil &&
			(strings.HasSuffix(path, index.ItemKindSegment) || strings.HasSuffix(path, index.ItemKindSnapshot)) {
			sectionName := bluge_directory_elf.SectionPrefixForIndex(flag.Arg(1)) + filepath.Base(path)

			outputElfPath = flag.Arg(0) + "." + fmt.Sprintf("%d", i)
			err := addSection(inputElfPath, sectionName, path, outputElfPath)
			if err != nil {
				return fmt.Errorf("error adding section: %v", err)
			}
			if i > 0 {
				err = os.Remove(inputElfPath)
				if err != nil {
					return fmt.Errorf("error removing intermediate file '%s': %v", inputElfPath, err)
				}
			}
			inputElfPath = outputElfPath
			i++
		}

		return nil
	})
	if err != nil {
		log.Fatalf("error walking bluge index: %v", err)
	}

	if outputElfPath != "" {
		err = os.Rename(outputElfPath, flag.Arg(0)+".withindex")
		if err != nil {
			log.Fatalf("error renaming final output: %v", err)
		}
	}
}

func addSection(inputElfPath, sectionName, sectionDataPath, outputElfPath string) error {
	cmd := exec.Command("objcopy",
		"--add-section", sectionName+"="+sectionDataPath,
		"--set-section-flags", sectionName+"=noload,readonly",
		inputElfPath, outputElfPath)
	return cmd.Run()
}
