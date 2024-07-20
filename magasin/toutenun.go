package magasin

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"qsync/backend_api"
	"qsync/bdd"
	"qsync/filesystem"
	"qsync/globals"
	"runtime"
	"strings"

	"github.com/skratchdot/open-golang/open"
)

func RunInstaller(path string) {

	switch runtime.GOOS {
	case "linux":
		log.Println("asking root privileges to run installer")
		backend_api.RunDPKGAsRoot(path)
	case "windows":
		open.Run(path)
	}
}

func DownloadFromUrl(url string, installer_path string) error {
	out, err := os.Create(installer_path)
	if err != nil {
		return err
	}

	defer out.Close()

	resp, err := http.Get(url)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)

	if err != nil {
		return err
	}

	return nil
}

func InstallApp(data io.ReadCloser) error {
	var json_data globals.ToutEnUnConfig
	err := json.NewDecoder(data).Decode(&json_data)

	log.Println("installing app : ")
	log.Println(json_data)

	// by default, the app will be installed to <qsync_installation_root>/apps/<app_name>

	if err != nil {
		return err
	}

	// getting qsync root path
	self_path, err := os.Executable()
	if err != nil {
		log.Fatal("An error occured while determining the path to qsync executable in InstallApp()", err)
	}

	self_path = filepath.Dir(self_path)
	log.Println("qsync root path : ", self_path)

	apps_path := filepath.Join(self_path, "apps")
	ex := globals.Exists(apps_path)

	if !ex {
		os.Mkdir(apps_path, fs.ModePerm)
	}

	sanitized_app_name := strings.ReplaceAll(json_data.AppName, " ", "_")
	new_app_root_path := filepath.Join(apps_path, sanitized_app_name)
	json_data.AppLauncherPath = filepath.Join(new_app_root_path, json_data.AppLauncherPath)

	ex = globals.Exists(new_app_root_path)

	if !ex {
		os.Mkdir(new_app_root_path, fs.ModePerm)
	}

	if json_data.NeedsInstaller {
		// pre-determined installer name so there are no problem

		json_data.AppInstallerPath = strings.ReplaceAll(json_data.AppInstallerPath, " ", "_")

		err = DownloadFromUrl(json_data.AppDownloadUrl, json_data.AppInstallerPath)

		if err != nil {
			return err
		}

		if json_data.NeedsInstaller {
			log.Println("Running installer...")
			RunInstaller(json_data.AppInstallerPath)
		}
	} else {
		// still need to download portable executable

		err = DownloadFromUrl(json_data.AppDownloadUrl, json_data.AppLauncherPath)

		if err != nil {
			return err
		}
	}

	// and last but not least, if the installed did not create it, create the sync folder
	app_sync_folder := filepath.Join(new_app_root_path, json_data.AppSyncDataFolderPath)
	ex = globals.Exists(app_sync_folder)
	log.Println("making app sync directory : ", app_sync_folder)

	if !ex {
		os.Mkdir(app_sync_folder, fs.ModePerm)
	}

	if err != nil {
		return err
	}

	// finish by putting non relative path so we can revover them easily from the database
	// when launching app
	json_data.AppSyncDataFolderPath = filepath.Join(new_app_root_path, json_data.AppSyncDataFolderPath)
	json_data.AppInstallerPath = filepath.Join(new_app_root_path, json_data.AppInstallerPath)
	json_data.AppUninstallerPath = filepath.Join(new_app_root_path, json_data.AppUninstallerPath)

	// now its time to register in the database the new little app
	var acces bdd.AccesBdd

	acces.InitConnection()
	defer acces.CloseConnection()

	acces.CreateSync(app_sync_folder)

	acces.AddToutEnUn(&json_data)

	log.Println("added app to database")

	// start watching the app's derectory
	go filesystem.StartWatcher(app_sync_folder)

	backend_api.ShowAlert("Application " + json_data.AppName + " installed !")

	return nil

}

func UninstallToutenun(config globals.MinGenConfig) error {

	cmd := exec.Command(config.UninstallerPath)

	err := cmd.Run()
	return err
}
