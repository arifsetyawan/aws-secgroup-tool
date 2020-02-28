package main

import (
	"bytes"
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
	Description string `json:"description"`
}

type ListOfSecGrp []SecGrp

func main() {

	saveCmd := flag.NewFlagSet("save", flag.ExitOnError)
	saveGroupId := saveCmd.String("gid", "", "-gid=<your_group_id>")
	saveProfile := saveCmd.String("profile", "", "-profile=<your_group_id>")
	description := saveCmd.String("description", "", "-description=<your role description>")

	removeCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	removeGroupId := removeCmd.String("gid", "", "-gid=<your_group_id>")

	if len(os.Args) < 2 {
		os.Exit(1)
	}

	switch os.Args[1] {
	case "save":
		saveCmd.Parse(os.Args[2:])

		if saveGroupId == nil || saveProfile == nil || description == nil {
			fmt.Errorf("%v", "param not complete")
			break
			return
		}

		params := SecGrp{
			GroupId:*saveGroupId,
			Profile:*saveProfile,
			Description:*description,
		}
		mods("save", params)
	case "remove":
		removeCmd.Parse(os.Args[2:])
		params := SecGrp{
			GroupId:*removeGroupId,
		}
		mods("remove", params)
	case "list":
		list()
	case "white":
		performWitelisting()
	}

}

func mods(action string, params SecGrp) {
	var listOfGroupIds ListOfSecGrp

	if params.GroupId == "" {
		_, _ =fmt.Fprintln(os.Stderr, "group id must not empty")
		return
	}

	byteValue, _ := getFileContent("groupList.json")
	_  = json.Unmarshal(byteValue, &listOfGroupIds)

	if action == "save" {
		listOfGroupIds = append(listOfGroupIds, params)
	}

	if action == "remove" {
		var newListOfGroupIds ListOfSecGrp
		for _, val := range listOfGroupIds {
			if val.GroupId != params.GroupId {
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

	fmt.Println(action+" "+params.GroupId+" success")

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
		fmt.Println("gid:",group.GroupId,"profile:", group.Profile,"description:",group.Description)
	}
	return listOfGroup
}

func performWitelisting() bool {
	var err error
	var out bytes.Buffer
	var stderr bytes.Buffer

	binary, lookErr := exec.LookPath("aws")
	if lookErr != nil {
		return false
	}

	res, _ := http.Get("https://api.ipify.org")
	ip, _ := ioutil.ReadAll(res.Body)

	fmt.Println("current ip:", string(ip))
	lastIp := getLastSavedIp()
	fmt.Println("last ip:", lastIp)

	for _, group := range list() {
		if lastIp != "" {
			revokeCmd := exec.Command(binary, "ec2","revoke-security-group-ingress", "--group-id", group.GroupId, "--ip-permissions", permissionString(lastIp, group.Description),"--profile", group.Profile)
			revokeCmd.Stdout = &out
			revokeCmd.Stderr = &stderr
			err := revokeCmd.Run()
			if err != nil {
				fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			}
		}
		authCmd := exec.Command(binary, "ec2", "authorize-security-group-ingress", "--group-id", group.GroupId, "--ip-permissions",permissionString(string(ip), group.Description),"--profile", group.Profile)
		authCmd.Stdout = &out
		authCmd.Stderr = &stderr
		err := authCmd.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
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
	fmt.Println("addBytVal", string(byteValue))
	addr := net.ParseIP(string(byteValue))
	fmt.Println("addr", addr)
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

func permissionString(ip string, desc string) string {
	return fmt.Sprint("IpProtocol=tcp,FromPort=22,ToPort=22,IpRanges=[{CidrIp=",string(ip),"/32,Description=",desc,"}]")
}