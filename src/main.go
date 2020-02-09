package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
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
	case "white":
		performWitelisting()
	}

}

func mods(action string, groupId string, profile string) {
	var listOfGroupIds ListOfSecGrp

	if groupId == "" {
		_, _ =fmt.Fprintln(os.Stderr, "group id must not empty")
	}

	byteValue, _ := getFileContent("groupList.json")
	_  = json.Unmarshal(byteValue, &listOfGroupIds)

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
	errSaveNewList := writeFileContent("groupList.json", preparedNewList)
	if errSaveNewList != nil {
		_ = fmt.Errorf("%v", errSaveNewList)
		panic(errSaveNewList)
	}

	fmt.Println(action+" "+groupId+" success")

	list()

}

func list() ListOfSecGrp {
	var listOfGroup ListOfSecGrp

	byteValue, _ := getFileContent("groupList.json")
	_  = json.Unmarshal(byteValue, &listOfGroup)

	if len(listOfGroup) == 0 {
		fmt.Println("No List")
		return ListOfSecGrp{}
	}

	for _, group := range listOfGroup {
		fmt.Println("gid:",group.GroupId,"profile:", group.Profile)
	}
	return listOfGroup
}

func performWitelisting() bool {
	var err error

	binary, lookErr := exec.LookPath("aws")
	if lookErr != nil {
		return false
	}

	res, _ := http.Get("https://api.ipify.org")
	ip, _ := ioutil.ReadAll(res.Body)

	fmt.Println("current ip:", string(ip))
	lastIp := getLastSavedIp()

	for _, group := range list() {
		if lastIp != "" {
			_, err = exec.Command(binary, "ec2","revoke-security-group-ingress", "--group-id", group.GroupId, "--protocol", "tcp", "--port" ,"22", "--cidr", string(ip)+"/32","--profile", group.Profile).CombinedOutput()
		}
		_, err = exec.Command(binary, "ec2", "authorize-security-group-ingress", "--group-id", group.GroupId, "--protocol", "tcp", "--port" ,"22", "--cidr", string(ip)+"/32","--profile", group.Profile).CombinedOutput()
		if err != nil {
			fmt.Println("error cmd", err)
		}
	}

	if err != nil {
		_ = fmt.Errorf("Error Execute aws configure ", err)
		return false
	}

	errorWrite := writeFileContent("lastIp", []byte(ip))
	if errorWrite != nil {
		_ = fmt.Errorf("%v", errorWrite)
		return false
	}

	return true
}

func getLastSavedIp() string {
	byteValue, _ := getFileContent("lastIp")
	addr := net.ParseIP(string(byteValue))
	if addr == nil {
		return ""
	}
	return addr.String()
}

func getFileContent(filename string) ([]byte, error) {
	usr, err := user.Current()
	if err != nil {
		_ = fmt.Errorf("%v", err)
	}
	if _, err := os.Stat(usr.HomeDir+"/.awssecgroup/"); os.IsNotExist(err) {
		_ = os.Mkdir(usr.HomeDir+"/.awssecgroup", 0777)
	}
	theFile, err := os.Open(usr.HomeDir+"/.awssecgroup/"+filename)
	if err != nil {
		return nil, err
	}
	defer theFile.Close()
	return ioutil.ReadAll(theFile)
}

func writeFileContent(filename string, content []byte) error {
	usr, err := user.Current()
	if err != nil {
		_ = fmt.Errorf("%v", err)
	}
	if _, err := os.Stat(usr.HomeDir+"/.awssecgroup/"); os.IsNotExist(err) {
		_ = os.Mkdir(usr.HomeDir+"/.awssecgroup", 0777)
	}
	return ioutil.WriteFile(usr.HomeDir+"/.awssecgroup/"+filename, content, 0644)
}
