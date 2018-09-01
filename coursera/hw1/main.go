package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

type fileInfo struct {
	prefix string
	file   os.FileInfo
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, printFiles bool) (err error) {
	dir, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(out, err)
		os.Exit(1)
	}
	defer dir.Close()

	slice := getFileInfoList(dir, printFiles, "")
	printFormatedOutput(out, slice)
	return nil
}

func getFileInfoList(dir *os.File, printFiles bool, prefix string) (s []fileInfo) {
	files, err := dir.Readdir(-1)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if !printFiles {
		files = filterDirectories(files)
	}

	for idx, file := range files {
		s = append(s, fileInfo{prefix: prefix + getPrefix(idx, len(files)), file: file})

		if file.IsDir() {
			nextDir, err := os.Open(dir.Name() + string(os.PathSeparator) + file.Name())
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			defer nextDir.Close()

			directorySymbol := ""
			if idx != len(files)-1 {
				directorySymbol += "│"
			}
			nestedSlice := getFileInfoList(nextDir, printFiles, prefix+directorySymbol+"\t")
			s = append(s, nestedSlice...)
		}
	}

	return s
}

func filterDirectories(values []os.FileInfo) (ret []os.FileInfo) {
	for _, s := range values {
		if s.IsDir() {
			ret = append(ret, s)
		}
	}

	return ret
}

func printFormatedOutput(out io.Writer, s []fileInfo) {
	for _, v := range s {
		fmt.Fprintln(out, v.prefix+"───"+formatFileName(v.file))
	}
}

func getPrefix(currentIndex int, length int) (symbol string) {
	if currentIndex == (length - 1) {
		return "└"
	} else {
		return "├"
	}
}

func formatFileName(fileInfo os.FileInfo) string {
	if fileInfo.IsDir() {
		return fileInfo.Name()
	} else {
		return fileInfo.Name() + " " + formatFileSize(fileInfo.Size())
	}
}

func formatFileSize(size int64) string {
	if size == 0 {
		return "(empty)"
	} else {
		return "(" + strconv.FormatInt(size, 10) + "b)"
	}
}
