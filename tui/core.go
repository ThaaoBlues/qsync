package tui

import (
	"fmt"
	"log"
	"path/filepath"
	"qsync/backend_api"
	"qsync/bdd"
	"qsync/filesystem"
	"qsync/globals"
	"qsync/magasin"
	"qsync/networking"
	"strconv"
	"time"
)

var LOGO string = `
________/\\\___________/\\\\\\\\\\\_____________________________________________        
_____/\\\\/\\\\______/\\\/////////\\\___________________________________________       
 ___/\\\//\////\\\___\//\\\______\///_____/\\\__/\\\_____________________________      
  __/\\\______\//\\\___\////\\\___________\//\\\/\\\___/\\/\\\\\\_______/\\\\\\\\_     
   _\//\\\______/\\\_______\////\\\_________\//\\\\\___\/\\\////\\\____/\\\//////__    
    __\///\\\\/\\\\/___________\////\\\_______\//\\\____\/\\\__\//\\\__/\\\_________   
     ____\////\\\//______/\\\______\//\\\___/\\_/\\\_____\/\\\___\/\\\_\//\\\________  
      _______\///\\\\\\__\///\\\\\\\\\\\/___\//\\\\/______\/\\\___\/\\\__\///\\\\\\\\_ 
       _________\//////_____\///////////______\////________\///____\///_____\////////__`

var MENU string = `

[0] - Start QSync
[1] - Create a sync task
[2] - Link another machine to a sync task on yours
[3] - List current sync task and their id
[4] - List devices using qsync on your network
[5] - Open QSync Magasin
[6] - Send something to another device : "Largage Aérien"
`

var PROMPT string = "\n>> "

var PROCESSING_EVENT bool
var CURRENT_EVENT_FLAG string

func Prompt() string {

	// if we are here, no backend events have been detected
	// we can safely display the regular menu's prompt
	fmt.Print(PROMPT)
	var query string
	_, err := fmt.Scanln(&query)

	if err != nil && err.Error() != "unexpected newline" {
		log.Fatal("Error while reading user query in Prompt() : ", err)
	}

	if PROCESSING_EVENT {
		backend_api.GiveInput(CURRENT_EVENT_FLAG, query)
		query = "[BACKEND_OVERRIDE]"
		PROCESSING_EVENT = false
	}
	return query
}

func AskConfirmation(msg string, validation string) bool {
	fmt.Println(msg)
	return Prompt() == validation
}

func ClearTerm() {
	fmt.Print("\033[H\033[2J")
}

func HandleMenuQuery(query string) {

	var acces bdd.AccesBdd

	acces.InitConnection()

	switch query {

	case "0":

		fmt.Println(("Starting watcher ..."))
		tasks := acces.ListSyncAllTasks()
		for i := 0; i < tasks.Size(); i++ {
			filesystem.StartWatcher(tasks.Get(i).Path)
		}

	case "1":

		fmt.Println("Enter below the path of the folder you want to synchronize :")

		var path string = Prompt()

		acces.CreateSync(path)

		fmt.Println("Sync task created. It can be started with the others from the menu.")

	case "2":

		fmt.Println("Select below the sync task you want to provide for another device :")
		tasks := acces.ListSyncAllTasks()
		for i := 0; i < tasks.Size(); i++ {
			task := tasks.Get(i)
			fmt.Println("{")
			fmt.Println("Path : ", task.Path)
			fmt.Println("Secure id : ", task.SecureId)
			fmt.Println("}")
		}

		index, err := strconv.Atoi(Prompt())

		if err != nil {
			log.Fatal("An error occured while scanning for a integer in HandleMenuQuery() : ", err)
		}

		if index > tasks.Size() {
			log.Fatal("The number you provied was not corresponding to any task.")
		}

		acces.GetSecureIdFromRootPath(tasks.Get(index).Path)

		fmt.Println("Mapping available devices on your local network...")

		// list qsync devices across the network
		devices := networking.GetNetworkDevices()
		for i := 0; i < devices.Size(); i++ {
			fmt.Printf("[%d] ", i)
			fmt.Println(devices.Get(i))
		}

		// send a link device request to the one the user choose

		index, err = strconv.Atoi(Prompt())

		if err != nil {
			log.Fatal("An error occured while scanning for a integer in HandleMenuQuery() : ", err)
		}

		device_id := devices.Get(index)["device_id"]

		var event globals.QEvent
		event.Flag = "[LINK_DEVICE]"
		event.SecureId = acces.SecureId
		event.FilePath = tasks.Get(index).Path

		var queue globals.GenArray[globals.QEvent]
		queue = queue.Add(event)
		var device_ids globals.GenArray[string]
		device_ids = device_ids.Add(device_id)

		networking.SendDeviceEventQueueOverNetwork(device_ids, acces.SecureId, queue, devices.Get(index)["ip_addr"])

		// link the device into this db
		acces.LinkDevice(device_id, devices.Get(index)["ip_addr"])
		log.Println("device linked")

		log.Println("Press any key once you have put the destination path on your other machine.")
		Prompt()
		// build a custom queue so this device can download all the data contained in your folder
		networking.BuildSetupQueue(acces.SecureId, device_id)

		fmt.Println("The selected device has successfully been linked to a sync task.")

	case "3":
		tasks := acces.ListSyncAllTasks()
		for i := 0; i < tasks.Size(); i++ {
			task := tasks.Get(i)
			fmt.Println("{")
			fmt.Println("Path : ", task.Path)
			fmt.Println("Secure id : ", task.SecureId)
			fmt.Println("}")
		}

	case "4":
		// list qsync devices across the network

		devices := networking.GetNetworkDevices()
		for i := 0; i < devices.Size(); i++ {
			fmt.Printf("[%d] ", i)
			fmt.Println(devices.Get(i))
		}

	case "5":
		// open QSync store
		go magasin.StartServer()
		time.Sleep(1 * time.Second)
		magasin.OpenUrlInWebBrowser("http://127.0.0.1:8275")

	case "6":
		fmt.Println("Paste the absolute path to the file you want to send")
		filepath := Prompt()

		fmt.Println("Select a device on the network : ")
		devices := networking.GetNetworkDevices()
		for i := 0; i < devices.Size(); i++ {
			fmt.Printf("[%d] ", i)
			fmt.Println(devices.Get(i))
		}
		index, err := strconv.Atoi(Prompt())
		if err != nil || index > devices.Size() {
			log.Fatal("Vous n'avez pas saisi un nombre valide !")
		}

		fmt.Println("Sending " + filepath + " to " + devices.Get(index)["host"])
		networking.SendLargageAerien(filepath, devices.Get(index)["ip_addr"])

	case "[BACKEND_OVERRIDE]":
		break

	default:
		fmt.Println("This option does not exists :/")
		HandleMenuQuery(Prompt())
	}

}

func DisplayMenu() {

	fmt.Print(LOGO)
	fmt.Print(MENU)

	// interactive events callbacks
	callbacks := make(map[string]func(string))

	callbacks["[CHOOSELINKPATH]"] = func(context string) {
		// simulate new prompt as the real one is displayed before the text
		fmt.Print("\n" + context + "\n\n>> ")
		// don't give back response, as it is handled by the regular prompt-loop
		PROCESSING_EVENT = true
		CURRENT_EVENT_FLAG = "[CHOOSELINKPATH]"

		// wait user input in regular prompt system
		for PROCESSING_EVENT {
			time.Sleep(time.Millisecond * 500)
		}

		// let the backend process and suppress the event file

		time.Sleep(1 * time.Second)
	}

	// air dropping something
	callbacks["[OTDL]"] = func(context string) {
		// simulate new prompt as the real one is displayed before the text
		fmt.Print("\n" + context + "\n\n>> ")

		// don't give back response, as it is handled by the regular prompt-loop
		PROCESSING_EVENT = true
		CURRENT_EVENT_FLAG = "[OTDL]"

		// wait user input in regular prompt system
		for PROCESSING_EVENT {
			time.Sleep(time.Millisecond * 500)
		}

		fmt.Print("\nSaving file to the folder : " + filepath.Join(globals.QSyncWriteableDirectory, "largage_aerien") + "\n\n>> ")

		// let the backend process and suppress the event file
		time.Sleep(1 * time.Second)
	}

	go backend_api.WaitEventLoop(callbacks)

	for {
		HandleMenuQuery(Prompt())
	}

}
