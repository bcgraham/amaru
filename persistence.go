package main

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const EXT = ".amaru"
const BACKUP_DIR = "storage"

func init() {
	err := os.MkdirAll(BACKUP_DIR, os.ModeDir)
	if err != nil {
		log.Fatal(err)
	}
}

func (u *urlMap) backup() {
	u.Lock()
	defer u.Unlock()
	if len(u.unsaved) < 1 {
		return
	}
	go writeBackup(u.unsaved, filepath.Join(BACKUP_DIR, strconv.Itoa(int(time.Now().Unix()))))
	u.unsaved = make([]string, 0)
}

func writeBackup(unsaved []string, filename string) {
	f, err := os.Create(filename + EXT)
	if err != nil {
		log.Fatal("amaru: problem creating file: %v\n", err)
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	b := make(map[string]string)
	for _, token := range unsaved {
		b[token] = um.urls[token]
	}
	err = enc.Encode(b)
	if err != nil {
		log.Fatal("amaru: problem encoding b: %v\n", err)
	}
}

func (u urlMap) loadHistory() (last int64, err error) {
	dir, err := os.Getwd()
	if err != nil {
		return -1, fmt.Errorf("amaru: error getting current directory")
	}
	dir = filepath.Join(dir, BACKUP_DIR)
	listing, err := ioutil.ReadDir(dir)
	if err != nil {
		return -1, fmt.Errorf("amaru: error reading current directory")
	}
	for _, info := range listing {
		if !strings.HasSuffix(info.Name(), EXT) || info.IsDir() {
			continue
		}
		local_last, err := u.load(filepath.Join(dir, info.Name()))
		if err != nil {
			log.Printf("amaru: error loading backup %v: %v; continuing with next file...\n", info.Name(), err)
		}
		if local_last > last {
			last = local_last
		}
	}
	return last, nil
}

func (u urlMap) load(filename string) (last int64, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return -1, fmt.Errorf("Could not open %v", filename)
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	backup := make(map[string]string)
	err = dec.Decode(&backup)
	if err != nil {
		return -1, fmt.Errorf("Could not decode contents of %v", filename)
	}
	for short, long := range backup {
		if _, ok := u.urls[short]; ok {
			log.Printf("When loading a backup (%v), trying to add a token that already exists in our reconstructed urlmap (%v); continuing with next key/value pair...\n", filename, short)
			continue
		}
		u.urls[short] = long
		shortInt, err := strconv.ParseInt(short, 32, 64)
		if err != nil {
			log.Printf("When loading a backup (%v), error converting token to int64: %v\n", filename, short)
			continue
		}
		if shortInt > last {
			last = shortInt
		}
	}
	return last, nil
}
