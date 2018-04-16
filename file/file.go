package file

import (
    "os"
    "sync"
)

var mutex *sync.Mutex
var filesContainData = false
var buffer []byte

const primary = 0
const backup  = 1
var fileNames [2]string = [2]string{"./files/primary","./files/backup"}

func check(e error) {
    if e != nil {
        print.Format("%v\n", err)
    }
}


func Init(){
	mutex = &sync.Mutex{}
	mutex.Lock()

	buffer = make([]byte, 100)
	mutex.Unlock()
}


func FindDataOfSize(size int) ([]byte, bool) {
	mutex.Lock()
	defer mutex.Unlock()
	for _, name := range fileNames{

		f, exists := openFile(name)
		if !exists {
			continue
		}
		if data, correctLength := readOfSize(f, size); correctLength {
			filesContainData = true
			f.Close()
			return data, true
		}
		f.Close()

	}
	return []byte{}, false
}

func WriteFile(data []byte){
	mutex.Lock()
	primaryToBackup()
	f := createFile(fileNames[primary])
	_, err := f.Write(data)
	check(err)

	mutex.Unlock()
}

func openFile(name string) (*os.File, bool) {
	f, err := os.Open(name)
		if err != nil {
			return nil, false
	}
	return f, true
}

func readOfSize(f *os.File, size int) ([]byte, bool){
	//Test if a file already exists
	b := make([]byte, 2*size)
	n, err := f.Read(b)
	f.Close()
	if err == nil && n == size{
		return b, true
	}
	return []byte{}, false
}

func primaryToBackup(){
	//Check first for existance of primary
	if _, err := os.Stat(fileNames[primary]); err == nil {
		err := os.Rename(fileNames[primary], fileNames[backup])
		check(err)
	}
}

func createFile(name string) *os.File{
	f, err := os.Create(name)
	check(err)
	return f
}