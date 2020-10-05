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

package bluge_directory_elf

import (
	"debug/elf"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/blevesearch/mmap-go"
	"github.com/blugelabs/bluge/index"
	segment "github.com/blugelabs/bluge_segment_api"
)

const sectionPrefix = "bluge/"

func SectionPrefixForIndex(name string) string {
	return sectionPrefix + name + "/"
}

type ElfDirectory struct {
	elfPath          string
	sectionPrefix    string
	segments         uint64Slice
	snapshots        uint64Slice
	segmentSections  map[uint64]*elf.Section
	snapshotSections map[uint64]*elf.Section
}

func NewElfDirectory(elfPath, dirName string) *ElfDirectory {
	return &ElfDirectory{
		elfPath:          elfPath,
		sectionPrefix:    SectionPrefixForIndex(dirName),
		segmentSections:  make(map[uint64]*elf.Section),
		snapshotSections: make(map[uint64]*elf.Section),
	}
}

func (e *ElfDirectory) Setup(readOnly bool) (err error) {
	if !readOnly {
		return fmt.Errorf("elf directoy does not support read-write")
	}

	elfFile, err := elf.Open(e.elfPath)
	if err != nil {
		return fmt.Errorf("error opening elf file: %v", err)
	}
	defer func() {
		cerr := elfFile.Close()
		if err == nil {
			err = cerr
		}
	}()

	for _, section := range elfFile.Sections {
		if strings.HasPrefix(section.Name, e.sectionPrefix) {
			rest := strings.TrimPrefix(section.Name, e.sectionPrefix)
			if strings.HasSuffix(rest, index.ItemKindSegment) {
				segmentString := strings.TrimSuffix(rest, index.ItemKindSegment)
				segmentNumber, err := strconv.ParseUint(segmentString, 16, 64)
				if err != nil {
					return fmt.Errorf("invalid segment number '%s': %v", segmentString, err)
				}
				e.segments = append(e.segments, segmentNumber)
				e.segmentSections[segmentNumber] = section
			} else if strings.HasSuffix(rest, index.ItemKindSnapshot) {
				snapshotString := strings.TrimSuffix(rest, index.ItemKindSnapshot)
				snapshotEpoch, err := strconv.ParseUint(snapshotString, 16, 64)
				if err != nil {
					return fmt.Errorf("invalid snapshot epoch '%s': %v", snapshotString, err)
				}
				e.snapshots = append(e.snapshots, snapshotEpoch)
				e.snapshotSections[snapshotEpoch] = section
			}
		}

	}

	sort.Sort(sort.Reverse(e.segments))
	sort.Sort(sort.Reverse(e.snapshots))

	return nil
}

func (e *ElfDirectory) List(kind string) ([]uint64, error) {
	if kind == index.ItemKindSegment {
		return e.segments, nil
	} else if kind == index.ItemKindSnapshot {
		return e.snapshots, nil
	}
	return nil, nil
}

func (e *ElfDirectory) Load(kind string, id uint64) (*segment.Data, io.Closer, error) {
	var section *elf.Section
	if kind == index.ItemKindSegment {
		section = e.segmentSections[id]
	} else if kind == index.ItemKindSnapshot {
		section = e.snapshotSections[id]
	}
	if section == nil {
		return nil, nil, fmt.Errorf("no such %s with id %d", kind, id)
	}

	f, err := os.OpenFile(e.elfPath, os.O_RDONLY, 0)
	if err != nil {
		return nil, nil, err
	}

	// we can only mmap on os pagesize boundaries
	pageStart, extraBytes := findNearestPage(section.Offset)

	mm, err := mmap.MapRegion(f, int(section.FileSize)+extraBytes, mmap.RDONLY, 0, pageStart)
	if err != nil {
		// mmap failed, try to close the file
		_ = f.Close()
		return nil, nil, err
	}

	closeFunc := func() error {
		err := mm.Unmap()
		// try to close file even if unmap failed
		err2 := f.Close()
		if err == nil {
			// try to return first error
			err = err2
		}
		return err
	}

	return segment.NewDataBytes(mm[extraBytes:]), closerFunc(closeFunc), nil
}

func findNearestPage(offset uint64) (pageStart int64, extraBytes int) {
	extraBytes = int(offset) % os.Getpagesize()
	return int64(offset) - int64(extraBytes), extraBytes
}

func (e *ElfDirectory) Stats() (numItems uint64, numBytes uint64) {
	return 0, 0
}

func (e *ElfDirectory) Persist(kind string, id uint64, w index.WriterTo, closeCh chan struct{}) error {
	return fmt.Errorf("elf directoy does not support read-write")
}

func (e *ElfDirectory) Remove(kind string, id uint64) error {
	return fmt.Errorf("elf directoy does not support read-write")
}

func (e *ElfDirectory) Sync() error {
	return fmt.Errorf("elf directoy does not support read-write")
}

func (e *ElfDirectory) Lock() error {
	return fmt.Errorf("elf directoy does not support read-write")
}

func (e *ElfDirectory) Unlock() error {
	return fmt.Errorf("elf directoy does not support read-write")
}

type uint64Slice []uint64

func (e uint64Slice) Len() int           { return len(e) }
func (e uint64Slice) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e uint64Slice) Less(i, j int) bool { return e[i] < e[j] }

type closerFunc func() error

func (c closerFunc) Close() error {
	return c()
}
