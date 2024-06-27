package networking

import (
	"log"
	"net"
	"os"
	"qsync/bdd"
	"qsync/globals"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

type ZeroConfService struct {
	Server *mdns.Server
}

func (zcs *ZeroConfService) Browse() {

	var acces bdd.AccesBdd
	acces.InitConnection()
	defer acces.CloseConnection()

	// check if we want to just browse the network or actually update a table
	old_connected_devices := acces.GetOnlineDevices()
	new_connected_devices := GetNetworkDevices()

	// first, we put all linked devices state to false
	for i := 0; i < old_connected_devices.Size(); i++ {
		acces.SetDeviceConnectionState(old_connected_devices.Get(i), false)
	}

	acces.CleanNetworkMap()
	// then, we put all linked and connected devices state to true
	for i := 0; i < new_connected_devices.Size(); i++ {

		new_device := new_connected_devices.Get(i)
		if !acces.IsDeviceOnNetworkMap(new_device["ip_addr"]) {
			acces.AddDeviceToNetworkMap(new_device["device_id"], new_device["ip_addr"], new_device["host"])
		}

		if acces.IsDeviceLinked(new_device["device_id"]) {
			log.Println("Detected device : ", new_device)
			acces.SetDeviceConnectionState(new_device["device_id"], true, new_device["ip_addr"])
			log.Println("Checking if he missed any updates : ")

			if acces.NeedsUpdate(new_device["device_id"]) {

				// this function returns a map with secure_id from tasks as keys
				// and the event queue from the sync task associated with the id as value.
				// the map only has the tasks the new device needs to catch up on
				multi_queue := acces.BuildEventQueueFromRetard(new_device["device_id"])

				for secure_id, ptr_queue := range multi_queue {
					acces.SecureId = secure_id

					// rebuild a queue of actual values and not pointers
					// HAHAH THIS IS NOT EFFICIENT AT ALL I WILL BURN THE WORLLDDDD
					var queue globals.GenArray[globals.QEvent]
					for i := 0; i < ptr_queue.Size(); i++ {
						ptr_event := ptr_queue.Get(i)
						log.Println(*ptr_event)

						queue.Add(*ptr_event)
					}

					var device_ids globals.GenArray[string]
					device_ids.Add(new_device["device_id"])
					SendDeviceEventQueueOverNetwork(device_ids, acces.SecureId, queue, new_device["ip_addr"])

				}

				acces.RemoveDeviceFromRetard(new_device["device_id"])
			}

		}
	}

	// so now, we have a fully up to date device table without making double loops or shits like that :D
}

func (zcs *ZeroConfService) UpdateDevicesConnectionStateLoop() {
	for {
		zcs.Browse()
		time.Sleep(10 * time.Second)
	}
}

func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func (zcs *ZeroConfService) Register() {

	//var err error

	var acces bdd.AccesBdd
	acces.InitConnection()

	// Setup our service export
	host, _ := os.Hostname()
	info := []string{
		"version=0.0.1-PreAlpha",
		"device_id=" + acces.GetMyDeviceId(),
	}
	service, _ := mdns.NewMDNSService(host, "_qsync._tcp.", "", "", 8274, []net.IP{GetOutboundIP()}, info)

	// Create the mDNS server, defer shutdown
	server, _ := mdns.NewServer(&mdns.Config{Zone: service})

	zcs.Server = server
}

func (zcs *ZeroConfService) Shutdown() {
	zcs.Server.Shutdown()
}

func GetNetworkDevices() globals.GenArray[map[string]string] {

	var devices_list globals.GenArray[map[string]string]

	// Make a channel for results and start listening
	entriesCh := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for entry := range entriesCh {

			dev := make(map[string]string, 0)

			dev["host"] = entry.Host + "local"
			dev["ip_addr"] = entry.AddrV4.String()
			// as the order of the supplementary fields seem to vary from
			// a service implementation to another

			// avoid malformed fields
			if len(strings.Split(entry.InfoFields[0], "=")) > 1 {
				if strings.Split(entry.InfoFields[0], "=")[0] == "version" {
					dev["version"] = strings.Split(entry.InfoFields[0], "=")[1]
					dev["device_id"] = strings.Split(entry.InfoFields[1], "=")[1]
				} else {
					dev["version"] = strings.Split(entry.InfoFields[1], "=")[1]
					dev["device_id"] = strings.Split(entry.InfoFields[0], "=")[1]
				}
				devices_list.Add(dev)
			}

		}
	}()

	// Start the lookup
	//log.SetOutput(io.Discard)
	mdns.Lookup("_qsync._tcp.", entriesCh)
	close(entriesCh)

	return devices_list
}
