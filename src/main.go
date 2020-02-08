package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
)

type SecGrp struct {
	GroupId string `json:"groupId"`
	Profile string `json:"profile"`
}

type ListOfSecGrp []SecGrp

func main() {

	saveCmd := flag.NewFlagSet("save", flag.ExitOnError)
	saveGroupId := saveCmd.String("gid", "", "-gid=<your_group_id>")
	saveProfile := saveCmd.String("profile", "", "-profile=<your_group_id>")

	removeCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	removeGroupId := removeCmd.String("gid", "", "-gid=<your_group_id>")

	if len(os.Args) < 2 {
		os.Exit(1)
	}

	switch os.Args[1] {
	case "save":
		saveCmd.Parse(os.Args[2:])
		mods("save", *saveGroupId, *saveProfile)
	case "remove":
		removeCmd.Parse(os.Args[2:])
		mods("remove", *removeGroupId, "")
	case "list":
		list()
	case "set":

		// Get Current User
		//usr, err := user.Current()
		//if err != nil {
		//	fmt.Errorf("%v", err)
		//}
		//
		//viper.SetConfigName("config") // name of config file (without extension)
		//viper.AddConfigPath(usr.HomeDir+"/.awsstsgen/")
		//err = viper.ReadInConfig()
		//if err != nil {
		//	panic(fmt.Errorf("Fatal error config file: %s \n", err))
		//}
		//
		//var mfaToken string
		//
		//// Get token prompt
		//fmt.Print("Please input current MFA token: ")
		//_, err = fmt.Scanln(&mfaToken)
		//if err != nil {
		//	panic("Require MFA Token")
		//}
		//
		//result, err := requestStsCredential(mfaToken, viper.Get("mfa-arn").(string), viper.Get("base-profile").(string))
		//if err != nil {
		//	fmt.Fprintf(os.Stderr, "error: %v\n", err)
		//	os.Exit(1)
		//}
		//
		//json.Unmarshal(result.([]byte), &theRes)
		//
		//credentials := theRes["Credentials"].(map[string]interface{})
		//accessKey := credentials["AccessKeyId"].(string)
		//secretKey := credentials["SecretAccessKey"].(string)
		//sessionToken := credentials["SessionToken"].(string)
		//
		//profile := viper.Get("target-profile").(string)
		//
		//fmt.Println("accessKey: "+accessKey)
		//fmt.Println("secretKey: "+secretKey)
		//fmt.Println("sessionToken: "+sessionToken)
		//
		//if configureLocalCredential(accessKey, secretKey, sessionToken, profile) == true {
		//	fmt.Println("\nCREDENTIAL UPDATED")
		//}
	}

}

func requestStsCredential(token string, serial string, baseProfile string) (interface{}, error) {
	binary, lookErr := exec.LookPath("aws")
	if lookErr != nil {
		return nil, lookErr
	}

	cmd := exec.Command(binary, "sts", "get-session-token", "--serial-number", serial, "--token-code", token, "--duration-seconds", "129600", "--profile", baseProfile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func configureLocalCredential(accessKey string, secretKey string, sessionToken string, profile string) bool {
	binary, lookErr := exec.LookPath("aws")
	if lookErr != nil {
		return false
	}
	_, err := exec.Command(binary, "configure", "set", "aws_access_key_id", accessKey, "--profile", profile).CombinedOutput()
	_, err = exec.Command(binary, "configure", "set", "aws_secret_access_key", secretKey, "--profile", profile).CombinedOutput()
	_, err = exec.Command(binary, "configure", "set", "aws_session_token", sessionToken, "--profile", profile).CombinedOutput()
	if err != nil {
		_ = fmt.Errorf("Error Execute aws configure ", err)
		return false
	}

	return true
}

func mods(action string, groupId string, profile string) {
	var listOfGroupIds ListOfSecGrp

	if groupId == "" {
		_, _ =fmt.Fprintln(os.Stderr, "group id must not empty")
	}

	usr, err := user.Current()
	if err != nil {
		_ = fmt.Errorf("%v", err)
		return
	}

	if _, err := os.Stat(usr.HomeDir+"/.awssecgroup/"); os.IsNotExist(err) {
		_ = os.Mkdir(usr.HomeDir+"/.awssecgroup", 0777)
	} else {

		jsonFile, err := os.Open(usr.HomeDir+"/.awssecgroup/groupList.json")
		if err != nil {
			_ = fmt.Errorf("%v", err)
		} else {
			byteValue, _ := ioutil.ReadAll(jsonFile)
			_  = json.Unmarshal(byteValue, &listOfGroupIds)
			defer jsonFile.Close()
		}
	}

	if action == "save" {
		listOfGroupIds = append(listOfGroupIds, SecGrp{GroupId:groupId, Profile:profile})
	}

	if action == "remove" {
		var newListOfGroupIds ListOfSecGrp
		for _, val := range listOfGroupIds {
			if val.GroupId != groupId {
				newListOfGroupIds = append(newListOfGroupIds, val)
			}
		}

		if len(listOfGroupIds) == len(newListOfGroupIds) {
			fmt.Println("no such record to delete")
			return
		}

		listOfGroupIds = newListOfGroupIds
	}

	preparedNewList, _ := json.Marshal(listOfGroupIds)

	errSaveNewList := ioutil.WriteFile(usr.HomeDir+"/.awssecgroup/groupList.json", preparedNewList, 0644)
	if errSaveNewList != nil {
		_ = fmt.Errorf("%v", errSaveNewList)
		panic(errSaveNewList)
	}

	fmt.Println(action+" "+groupId+" success")

	list()

}

func list() {

	var listOfGroupIds ListOfSecGrp

	usr, err := user.Current()
	if err != nil {
		_ = fmt.Errorf("%v", err)
	}

	if _, err := os.Stat(usr.HomeDir+"/.awssecgroup/"); os.IsNotExist(err) {
		_ = os.Mkdir(usr.HomeDir+"/.awssecgroup", 0777)
		fmt.Println("No List")
	}

	jsonFile, err := os.Open(usr.HomeDir+"/.awssecgroup/groupList.json")
	if err != nil {
		fmt.Println("No List")
		return
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	_  = json.Unmarshal(byteValue, &listOfGroupIds)

	if len(listOfGroupIds) == 0 {
		fmt.Println("No List")
		return
	}

	for _, group := range listOfGroupIds {
		fmt.Println("gid:",group.GroupId,"profile:", group.Profile)
	}
	return


}