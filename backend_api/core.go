package backend_api

import (
	"log"
	"os"
	"path/filepath"
	"qsync/globals"
	"time"

	"github.com/gen2brain/beeep"
)

func NotifyDesktop(msg string) {
	err := beeep.Alert("Qsync", msg, "assets/warning.png")
	if err != nil {
		panic(err)
	}
}

// IF THE BACKEND IS MULTITHREADED, DO NOT CLOSE APP BEFORE THE USER INPUT HAS BEEN
// PROCESSED BY BACKEND, THIS FUNCTION DOES NOT MAKES SURE OF IT
func AskInput(flag string, context string) string {

	f, err := os.Create(filepath.Join(globals.QSyncWriteableDirectory, flag+".btf"))
	defer os.Remove(filepath.Join(globals.QSyncWriteableDirectory, flag+".btf"))

	if err != nil {
		log.Fatal("Unable to Create input file in AskInput() : ", err)
	}

	f.WriteString(context)

	og_fstat, err_2 := os.Stat(filepath.Join(globals.QSyncWriteableDirectory, flag+".btf"))

	if err_2 != nil {
		log.Fatal("Unable to read stats of input file in AskInput() : ", err)
	}

	// wait for front-end to provide user input
	var nw_fstat os.FileInfo
	nw_fstat, err_2 = os.Stat(filepath.Join(globals.QSyncWriteableDirectory, flag+".btf"))

	if err_2 != nil {
		log.Fatal("Unable to read stats of input file in AskInput() : ", err)
	}

	for nw_fstat.Size() == og_fstat.Size() {
		time.Sleep(2 * time.Second)

		nw_fstat, err_2 = os.Stat(filepath.Join(globals.QSyncWriteableDirectory, flag+".btf"))

		if err_2 != nil {
			log.Fatal("Unable to read stats of input file in AskInput() : ", err)
		}

	}

	// now that we have the user input in the file, we can read it

	ret, err := os.ReadFile(filepath.Join(globals.QSyncWriteableDirectory, flag+".btf"))
	if err != nil {
		log.Fatal("Unable to Read input file in AskInput() : ", err)
	}
	return string(ret[len([]byte(context)):])
}

// use this function to get the message from the backend
// that is riding with the ask of user input
// must be used before providing the user's input
func ReadInputContext(flag string) string {

	buff, err := os.ReadFile(filepath.Join(globals.QSyncWriteableDirectory, flag+".btf"))
	if err != nil {
		log.Fatal("Unable to Read input file in ReadInputContext() : ", err)
	}

	return string(buff)
}

func GiveInput(flag string, data string) {
	f, err := os.OpenFile(filepath.Join(globals.QSyncWriteableDirectory, flag+".btf"), os.O_RDWR|os.O_APPEND, os.ModeAppend)
	if err != nil {
		log.Fatal("Unable to Create input file in AskInput() : ", err)
	}

	defer f.Close()

	f.WriteString(data)

}

func WaitEventLoop(callbacks map[string]func(context string)) {

	for {
		// Read the contents of the root directory
		files, err := os.ReadDir(globals.QSyncWriteableDirectory)
		if err != nil {
			log.Fatal("Error while reading directory in WaitEventLoop() : ", err)
		}

		// Check each file to see if it has a .btf extension
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if file.Name()[len(file.Name())-4:] == ".btf" {

				event_flag := file.Name()[:len(file.Name())-4]
				context_buff, err := os.ReadFile(file.Name())
				if err != nil {
					log.Fatal("Error while reading event fie in WaitEventLoop() : ", err)
				}

				callbacks[event_flag](string(context_buff))

			}
		}

		// Sleep for 1 second before checking again
		time.Sleep(1 * time.Second)
	}

}
