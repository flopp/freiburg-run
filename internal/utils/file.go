package utils

import "os"

func MakeDir(dir string) error {
	if err := os.MkdirAll(dir, 0770); err != nil {
		return err
	}
	return nil
}

func MustMakeDir(dir string) {
	err := MakeDir(dir)
	if err != nil {
		panic(err)
	}
}
