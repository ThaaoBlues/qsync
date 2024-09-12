package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var currentLang = "en" // Default language
var translations = map[string]map[string]string{
	"en": {
		"devicesTitle":        "Devices on your network",
		"tasksTitle":          "Active tasks on your device",
		"taskActions":         "Task Actions",
		"deviceActions":       "Device Actions",
		"sendFile":            "Send File",
		"sendFolder":          "Send Folder",
		"sendText":            "Send Text",
		"removeTask":          "Remove Task",
		"openApp":             "Open App",
		"syncAnotherDevice":   "Sync Another Device",
		"enableBackupMode":    "Enable Backup Mode",
		"disableBackupMode":   "Disable Backup Mode",
		"alertTaskCreated":    "Task Created at ",
		"navigationHelp":      "To navigate in Ecosys use the tab (\u2B7E) key to change section and the arrow up/down keys to select an option in the section. The enter key (\u23CE) is used to validate your selection.",
		"loading":             "Loading...",
		"createSyncTask":      "Create a new sync task",
		"openMagasin":         "Open the app marketplace",
		"toggleLargageAerien": "Allow/Refuse to receive Largages Aeriens",
		"openLargagesFolder":  "Open the folder where are stored received Largages Aeriens",
		"qToQuit":             "Press 'q' to get out of this section.",
	},
	"fr": {
		"devicesTitle":        "Appareils sur votre réseau",
		"tasksTitle":          "Tâches actives sur votre appareil",
		"taskActions":         "Actions de tâche",
		"deviceActions":       "Actions de l'appareil",
		"sendFile":            "Envoyer un fichier",
		"sendFolder":          "Envoyer un dossier",
		"sendText":            "Envoyer un texte",
		"removeTask":          "Supprimer la tâche",
		"openApp":             "Ouvrir l'application",
		"syncAnotherDevice":   "Synchroniser un autre appareil",
		"enableBackupMode":    "Activer le mode de sauvegarde",
		"disableBackupMode":   "Désactiver le mode de sauvegarde",
		"alertTaskCreated":    "Tâche créée à ",
		"navigationHelp":      "La navigation dans Ecosys se fait via la touche tab (\u2B7E) pour changer de section et les flèches haut/bas pour selectionner une option de la section. La touche entrée (\u23CE) est là pour valider.",
		"loading":             "Chargement...",
		"createSyncTask":      "Créer une tâche de synchronisation",
		"openMagasin":         "Ouvrir le magasin d'applications",
		"toggleLargageAerien": "Autoriser/Refuser les largages aerien",
		"openLargagesFolder":  "Ouvrir le dossier contenant les largages aerien",
		"qToQuit":             "Appuyez sur la touche 'q' pour sortir d'ici.",
	},
}

type Config struct {
	AppName            string   `json:"AppName"`
	AppDescription     string   `json:"AppDescription"`
	AppIconURL         string   `json:"AppIconURL"`
	SupportedPlatforms []string `json:"SupportedPlatforms"`
}

type Data struct {
	ToutEnUnConfigs []Config `json:"tout_en_un_configs"`
	GrapinConfigs   []Config `json:"grapin_configs"`
}

func updateLanguage(lang string) {
	currentLang = lang
}

func fetchTasks() []map[string]string {
	resp, err := http.Get("http://127.0.0.1:8275/list-tasks")
	if err != nil {
		log.Println("Error fetching tasks:", err)
		return []map[string]string{}
	}
	defer resp.Body.Close()

	var tasks []map[string]string
	json.NewDecoder(resp.Body).Decode(&tasks)
	return tasks
}

func fetchDevices() []map[string]string {
	resp, err := http.Get("http://127.0.0.1:8275/list-devices")
	if err != nil {
		log.Println("Error fetching devices:", err)
		return []map[string]string{}
	}
	defer resp.Body.Close()

	var devices []map[string]string
	json.NewDecoder(resp.Body).Decode(&devices)
	return devices
}

func CreateUI(app *tview.Application) tview.Primitive {

	updateLanguage("fr")

	// Header and Titles
	header := tview.NewTextView().
		SetText(`		
 ____  ___  __   ____  _  _  ____ 
(  __)/ __)/  \ / ___)( \/ )/ ___)
 ) _)( (__(  O )\___ \ )  / \___ \
(____)\___)\__/ (____/(__/  (____/
		`).
		SetTextColor(tcell.ColorGreen).
		SetTextAlign(tview.AlignCenter)

	devicesTitle := tview.NewTextView().
		SetText(translations[currentLang]["devicesTitle"]).
		SetTextColor(tcell.ColorYellow).
		SetDynamicColors(true)

	tasksTitle := tview.NewTextView().
		SetText(translations[currentLang]["tasksTitle"]).
		SetTextColor(tcell.ColorYellow).
		SetDynamicColors(true)

	menu := tview.NewFlex().SetDirection(tview.FlexRow)

	mainLayout := tview.NewFlex()

	// Menu buttons (navigable)
	createTaskBtn := tview.NewButton(translations[currentLang]["createSyncTask"]).SetSelectedFunc(func() {
		createSyncTask()
	})
	openMagasinBtn := tview.NewButton(translations[currentLang]["openMagasin"]).SetSelectedFunc(func() {
		openMagasin(app, mainLayout)
	})
	toggleLargageBtn := tview.NewButton(translations[currentLang]["toggleLargageAerien"]).SetSelectedFunc(func() {
		toggleLargageAerien()
	})
	openLargagesFolderBtn := tview.NewButton(translations[currentLang]["openLargagesFolder"]).SetSelectedFunc(func() {
		openLargagesFolder()
	})

	// Button list
	buttons := []*tview.Button{createTaskBtn, openMagasinBtn, toggleLargageBtn, openLargagesFolderBtn}
	menu = menu.
		AddItem(createTaskBtn, 2, 1, true).
		AddItem(openMagasinBtn, 2, 1, true).
		AddItem(toggleLargageBtn, 2, 1, true).
		AddItem(openLargagesFolderBtn, 2, 1, true)

	// Device List
	devicesList := tview.NewList()

	// Task List
	tasksList := tview.NewList()

	// Set up focus navigation between buttons
	for i, btn := range buttons {
		buttonIndex := i
		btn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyDown:
				app.SetFocus(buttons[(buttonIndex+1)%len(buttons)])
				return nil
			case tcell.KeyUp:
				app.SetFocus(buttons[(buttonIndex+len(buttons)-1)%len(buttons)])
				return nil
			}
			return event
		})
	}

	// Layout for devices and tasks
	devicesListLayout := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(devicesTitle, 1, 1, false).
			AddItem(devicesList, 0, 1, true), 0, 1, true)

	tasksListLayout := tview.NewFlex().AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tasksTitle, 1, 1, false).
		AddItem(tasksList, 0, 1, true), 0, 1, true)

	terminalParts := []*tview.Flex{menu, devicesListLayout, tasksListLayout}

	// Main layout
	mainLayout = mainLayout.
		SetDirection(tview.FlexRow).
		AddItem(header, 5, 1, false)

	// Add keyboard navigation between menu and content
	for i, layout := range terminalParts {
		layoutIndex := i
		layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			log.Println("Setting focus to another terminal part")

			switch event.Key() {
			case tcell.KeyTab:
				app.SetFocus(terminalParts[(layoutIndex+1)%len(terminalParts)])
				// remove highlight on previous focus zone
				terminalParts[layoutIndex].
					SetBorder(false)

				// highlight focus zone
				terminalParts[(layoutIndex+1)%len(terminalParts)].
					SetBorder(true).
					SetBorderStyle(tcell.StyleDefault).
					SetBorderColor(tcell.ColorGhostWhite)
			}
			return event
		})

		mainLayout = mainLayout.AddItem(layout, 0, 1, true)
	}

	// filling up lists

	go func() {
		for {
			app.QueueUpdateDraw(func() {
				devicesList.Clear()
				for _, device := range fetchDevices() {
					devicesList.AddItem(device["hostname"], device["ip_addr"], 0, func() {
						openDeviceActionsMenu(app, device, mainLayout)
					})
				}
			})
			time.Sleep(5 * time.Second)
		}
	}()

	go func() {
		for {
			app.QueueUpdateDraw(func() {
				tasksList.Clear()
				for _, task := range fetchTasks() {
					label := task["Path"]
					if task["IsApp"] == "true" {
						label = "( application ) " + task["Name"]
					}
					tasksList.AddItem(label, "", 0, func() {
						openTaskActionsMenu(app, task, mainLayout)
					})
				}
			})
			time.Sleep(5 * time.Second)
		}
	}()

	// affichage du panneau d'aide à l'utilisation de l'interface
	go func() {
		app.QueueUpdateDraw(func() {
			showNavigationHelp(app, mainLayout, translations[currentLang]["navigationHelp"])
		})
	}()

	return mainLayout
}

// Popup menus for task and device actions
func openTaskActionsMenu(app *tview.Application, task map[string]string, appRoot *tview.Flex) {
	var backupModeText string
	if task["BackupMode"] == "true" {
		backupModeText = translations[currentLang]["disableBackupMode"]
	} else {
		backupModeText = translations[currentLang]["enableBackupMode"]

	}

	modal := tview.NewModal().
		SetText(translations[currentLang]["taskActions"]).
		AddButtons([]string{
			translations[currentLang]["openApp"],
			translations[currentLang]["syncAnotherDevice"],
			translations[currentLang]["removeTask"],
			backupModeText,
		}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case translations[currentLang]["openApp"]:
				openApp(task)
			case translations[currentLang]["syncAnotherDevice"]:
				chooseDeviceAndLinkIt(app, task, appRoot)
			case translations[currentLang]["removeTask"]:
				removeTask(task)
			case translations[currentLang]["enableBackupMode"], translations[currentLang]["disableBackupMode"]:
				toggleBackupMode(task)
			}
			app.SetRoot(CreateUI(app), true)
		})
	app.SetRoot(modal, true).SetFocus(modal)
}

func openDeviceActionsMenu(app *tview.Application, device map[string]string, appRoot *tview.Flex) {
	modal := tview.NewModal().
		SetText(translations[currentLang]["deviceActions"]).
		AddButtons([]string{
			translations[currentLang]["sendFile"],
			translations[currentLang]["sendFolder"],
			translations[currentLang]["sendText"],
		}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case translations[currentLang]["sendFile"]:
				sendLargage(device, false)
				app.SetRoot(appRoot, true)

			case translations[currentLang]["sendFolder"]:
				sendLargage(device, true)
				app.SetRoot(appRoot, true)
			case translations[currentLang]["sendText"]:
				sendText(app, device, appRoot)
				// not setting root as sendText needs another form
			}
		})
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			log.Println("Hiding modal")
			modal.Blur()
			app.SetRoot(appRoot, true)
		}
		return event
	})

	app.SetRoot(modal, true).SetFocus(modal)
}

func createSyncTask() {
	_, err := http.Get("http://127.0.0.1:8275/create-task")
	if err != nil {
		fmt.Println("Error creating sync task:", err)
	}
}

func openMagasin(app *tview.Application, appRoot *tview.Flex) {
	pages := prepareMagasin(app, appRoot)
	app.SetRoot(pages, true)
}

func toggleLargageAerien() {
	_, err := http.Get("http://127.0.0.1:8275/toggle-largage")
	if err != nil {
		fmt.Println("Error toggling Largage Aerien:", err)
	}
}

func openLargagesFolder() {
	_, err := http.Get("http://127.0.0.1:8275/open-largages-folder")
	if err != nil {
		fmt.Println("Error opening Largages Folder:", err)
	}
}

func sendLargage(device map[string]string, folder bool) {

	// Request the file path from the user
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:8275/ask-file-path?is_folder=%t", folder))
	if err != nil {
		log.Println("Error fetching file path:", err)
		return
	}
	defer resp.Body.Close()

	var response map[string]string
	json.NewDecoder(resp.Body).Decode(&response)

	if response["Path"] != "[CANCELLED]" {

		data := map[string]interface{}{
			"filepath":  response["Path"],
			"device_id": device["device_id"],
			"ip_addr":   device["ip_addr"],
			"is_folder": folder,
		}

		// Send the largage
		jsonData, _ := json.Marshal(data)
		resp, err := http.Post("http://127.0.0.1:8275/send-largage", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Println("Error sending largage:", err)
		}

		//json.NewDecoder(resp.Body).Decode(&response)
		log.Println(resp.Body)
		defer resp.Body.Close()
	}
}

func sendText(app *tview.Application, device map[string]string, appRoot *tview.Flex) {
	form := tview.NewForm().
		AddTextArea("Text", "", 0, 20, 3000, nil)

	form.
		AddButton("Send", func() {
			text := form.GetFormItemByLabel("Text").(*tview.TextArea).GetText()
			// Send text
			data := map[string]interface{}{
				"device": device,
				"text":   text,
			}
			jsonData, _ := json.Marshal(data)
			resp, err := http.Post("http://127.0.0.1:8275/send-text", "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Println("Error sending text:", err)
			}
			defer resp.Body.Close()

			// Return to main layout after sending
			app.SetRoot(appRoot, true)
		}).
		AddButton("Cancel", func() {
			app.SetRoot(appRoot, true)
		})

	app.SetRoot(form, true).SetFocus(form)

}

func chooseDeviceAndLinkIt(app *tview.Application, task map[string]string, appRoot *tview.Flex) {
	// Fetch devices to display
	resp, err := http.Get("http://127.0.0.1:8275/list-devices")
	if err != nil {
		log.Println("Error fetching devices:", err)
		return
	}
	defer resp.Body.Close()

	var devices []map[string]string
	json.NewDecoder(resp.Body).Decode(&devices)

	list := tview.NewList()
	for _, device := range devices {
		d := device // Capture loop variable
		list.AddItem(device["hostname"], "", 0, func() {
			// Link the device with the task
			linkDevice(task, d)
			app.SetRoot(appRoot, true)
		})
	}

	// Set up modal for choosing device
	modal := tview.NewModal().
		SetText("Choose a device to synchronize").
		AddButtons(
			[]string{"Cancel"},
		).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(appRoot, true)
		})

	// Create a layout with the device list and the modal
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(list, 0, 1, true).
		AddItem(modal, 3, 1, false)

	app.SetRoot(layout, true).SetFocus(list)
}

func linkDevice(task map[string]string, device map[string]string) {
	data := map[string]string{
		"SecureId": task["SecureId"],
		"IpAddr":   device["ip_addr"],
		"DeviceId": device["device_id"],
	}

	// Send the request to link the device
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post("http://127.0.0.1:8275/link", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error linking device:", err)
	}
	defer resp.Body.Close()
}

func removeTask(task map[string]string) {
	_, err := http.Get(fmt.Sprintf("http://127.0.0.1:8275/remove-task?secure_id=%s", task["SecureId"]))
	if err != nil {
		log.Println("Error removing task:", err)
	}
}

func toggleBackupMode(task map[string]string) {
	_, err := http.Get(fmt.Sprintf("http://127.0.0.1:8275/toggle-backup-mode?secure_id=%s", task["SecureId"]))
	if err != nil {
		log.Println("Error toggling backup mode:", err)
	}
}

func openApp(task map[string]string) {
	_, err := http.Get(fmt.Sprintf("http://127.0.0.1:8275/launch-app?AppId=%s", task["SecureId"]))
	if err != nil {
		log.Println("Error launching app:", err)
	}
}

func prepareMagasin(app *tview.Application, appRoot *tview.Flex) *tview.Pages {
	pages := tview.NewPages()

	// Sections (ToutEnUn and Grapins)
	toutEnUnSection := tview.NewFlex().SetDirection(tview.FlexRow)
	grapinsSection := tview.NewFlex().SetDirection(tview.FlexRow)

	// indication pour sortir de la section
	toutEnUnSection.AddItem(tview.NewTextView().SetText(translations[currentLang]["qToQuit"]), 1, 0, false)
	grapinsSection.AddItem(tview.NewTextView().SetText(translations[currentLang]["qToQuit"]), 1, 0, false)

	// Add sections to Pages
	pages.AddPage("ToutEnUn", toutEnUnSection, true, true)
	pages.AddPage("Grapins", grapinsSection, true, true)

	menu := tview.NewList()

	// Fetch and process data
	go fetchMagasinData(app, pages, toutEnUnSection, grapinsSection)

	// Main menu to switch sections
	menu = menu.
		AddItem("Tout en un", "View Tout en un apps", 't', func() {
			pages.SwitchToPage("ToutEnUn")
		}).
		AddItem("Grapins", "View Grapins", 'g', func() {
			pages.SwitchToPage("Grapins")
		}).
		AddItem("Quit", translations[currentLang]["qToQuit"], 'q', func() {
			app.SetRoot(appRoot, true)
		})

	// Set menu as root of the pages
	pages.AddPage("Main", menu, true, true)
	// for some reason, without it the focus is on a non visible page
	pages.SendToBack("Main")

	return pages

}

// fetchData fetches the app configurations from the provided URL
func fetchMagasinData(app *tview.Application, pages *tview.Pages, toutEnUnSection *tview.Flex, grapinsSection *tview.Flex) {

	osName := findOsName()

	go showLoading(app, pages)

	// Fetch data
	url := "https://raw.githubusercontent.com/ThaaoBlues/ecosys/master/magasin_database.json"

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching config: %v", err)
	}
	defer resp.Body.Close()

	//test, err := os.Open("test_mag.json")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	var data Data
	if err := json.Unmarshal(body, &data); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	// List of cards for navigation
	var toutEnUnCards []*tview.Flex
	var grapinCards []*tview.Flex

	// Process Tout en un apps
	for _, config := range data.ToutEnUnConfigs {
		if contains(config.SupportedPlatforms, osName) {
			card := generateCard(config, "Install this app", func() {
				log.Println("Bouton installation cliqué")
				go showLoading(app, pages)
				go installApp(config, "/install-tout-en-un", app, pages)
			})
			app.QueueUpdateDraw(func() {
				toutEnUnSection = toutEnUnSection.AddItem(card, 15, 0, true)
			})

			toutEnUnCards = append(toutEnUnCards, card)

		}
	}

	// Process Grapin apps
	for _, config := range data.GrapinConfigs {
		if contains(config.SupportedPlatforms, osName) {
			card := generateCard(config, "Install this Grapin", func() {
				log.Println("Bouton installation cliqué")
				go showLoading(app, pages)
				go installApp(config, "/install-grapin", app, pages)
			})
			app.QueueUpdateDraw(func() {
				grapinsSection = grapinsSection.AddItem(card, 15, 0, true)
			})
			grapinCards = append(grapinCards, card)

		}
	}

	setupNavigation(app, pages, toutEnUnSection, toutEnUnCards)
	setupNavigation(app, pages, grapinsSection, grapinCards)

	// Hide loading popup after processing
	app.QueueUpdateDraw(func() {
		pages.SwitchToPage("Main")
		pages.RemovePage("Loading")
	})

}

// generateCard creates a card UI component for the app configuration
func generateCard(config Config, buttonText string, onClick func()) *tview.Flex {
	// Card layout
	card := tview.NewFlex().SetDirection(tview.FlexRow)

	// App title
	title := tview.NewTextView().SetText(config.AppName).SetDynamicColors(true).SetTextColor(tcell.ColorBlue)

	// App description
	description := tview.NewTextView().SetText(config.AppDescription).SetDynamicColors(true)

	// Install button
	button := tview.NewButton(buttonText).SetSelectedFunc(onClick)

	button.SetFocusFunc(func() {

		card.SetBorderStyle(tcell.StyleDefault)
		card.SetBorderColor(tcell.ColorGhostWhite)
		card.SetBorder(true)
	})

	button.SetBlurFunc(func() {
		card.SetBorder(false)
	})

	// Add components to the card
	card.AddItem(title, 3, 1, false).
		AddItem(button, 3, 1, true).
		AddItem(description, 4, 1, false)

	/*card.Focus(func(p tview.Primitive) {

		card.SetBorderStyle(tcell.StyleDefault)
		card.SetBorderColor(tcell.ColorGhostWhite)
		card.SetBorder(true)
	})

	card.SetBlurFunc(func() {
		card.SetBorder(false)

	})*/

	return card
}

// installApp performs an HTTP POST request to install the app
func installApp(config Config, endpoint string, app *tview.Application, pages *tview.Pages) {

	log.Println("Requète de l'installation de l'application au serveur web")
	url := fmt.Sprintf("http://127.0.0.1:8275%s", endpoint) // Adjust the URL as necessary

	// Marshal the app config into JSON
	jsonData, err := json.Marshal(config)
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}

	// Make the POST request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error installing app %s: %v", config.AppName, err)
		showError(app, pages, fmt.Sprintf("Error installing %s: %v", config.AppName, err))
		return
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response for app %s: %v", config.AppName, err)
		showError(app, pages, fmt.Sprintf("Error reading response for %s: %v", config.AppName, err))
		return
	}

	// Handle success
	log.Printf("Successfully installed app %s: %s", config.AppName, string(body))
	//showSuccess(app, pages, fmt.Sprintf("Successfully installed %s!", config.AppName))
}

// showLoading shows the loading modal
func showLoading(app *tview.Application, pages *tview.Pages) {
	loadingModal := tview.NewModal().
		SetText(translations[currentLang]["loading"]).
		AddButtons([]string{"Ok"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.SwitchToPage("Main")
			pages.RemovePage("Loading")
		})

	app.QueueUpdateDraw(func() {
		pages.AddPage("Loading", loadingModal, true, true)
		pages.SwitchToPage("Loading")
	})
}

// showError shows an error message in a modal
func showError(app *tview.Application, pages *tview.Pages, message string) {
	errorModal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Ok"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.SwitchToPage("Main")
			pages.RemovePage("Error")
		})

	app.QueueUpdateDraw(func() {
		pages.AddPage("Error", errorModal, true, true)
		pages.SwitchToPage("Error")
	})
}

// showSuccess shows a success message in a modal
func showSuccess(app *tview.Application, pages *tview.Pages, message string) {
	successModal := tview.NewModal()

	successModal.
		SetText(message).
		AddButtons([]string{"Ok"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.SwitchToPage("Main")
			successModal.Blur()
			pages.RemovePage("Success")
		})

	app.QueueUpdateDraw(func() {
		pages.AddPage("Success", successModal, true, true)
		pages.SwitchToPage("Success")
	})
}

// showNavigationHelp shows a help message to teach how to navigate in Ecosys in a modal
func showNavigationHelp(app *tview.Application, appRoot *tview.Flex, message string) {
	navigationHelpModal := tview.NewModal()

	navigationHelpModal.
		SetText(message).
		AddButtons([]string{"Ok"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			navigationHelpModal.Blur()
			app.SetRoot(appRoot, true)
		})

	app.SetRoot(navigationHelpModal, true).SetFocus(navigationHelpModal)
	navigationHelpModal.SetFocus(0)
}

// contains checks if a slice contains a specific item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func findOsName() string {
	return "Linux"
}

// setupNavigation enables arrow key navigation between cards
func setupNavigation(app *tview.Application, pages *tview.Pages, section *tview.Flex, cards []*tview.Flex) {

	l := len(cards)

	currentIndex := 0

	// Input capture for navigating between cards
	section.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp: // Navigate to the previous card
			currentIndex = (currentIndex + l - 1) % l
			app.SetFocus(cards[currentIndex].GetItem(1))
		case tcell.KeyDown: // Navigate to the next card
			currentIndex = (currentIndex + 1) % l
			app.SetFocus(cards[currentIndex].GetItem(1))

		}

		switch event.Rune() {
		case 'q':
			pages.SwitchToPage("Main")
		}
		return event
	})
}
