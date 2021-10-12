package uploader

import (
	"errors"
	"os/exec"
	"strings"
)

//return the instance id of this machine
//return "" for any error or not found
//currently support :'AWS'

func getAwsInstanceId() (string, error) {
	cmd := exec.Command("ec2-metadata", "-i")
	stdout, err := cmd.Output()
	if err != nil {
		return "", err
	}
	//e.g: instance-id: i-0528fe48372e54d50
	resultstr := string(stdout)
	idmix := strings.Split(resultstr, ":")
	if len(idmix) != 2 {
		return "", errors.New("id not exist:" + resultstr)
	} else {
		return strings.Trim(idmix[1], ""), nil
	}
}

func getAliInstanceId() (string, error) {
	//not implement yet
	return "", nil
}

func getAzureInstanceId() (string, error) {
	//not implement yet
	return "", nil
}

func GetInstanceId() string {
	instanceid, err := getAwsInstanceId()
	if instanceid != "" && err == nil {
		return instanceid
	}

	instanceid, err = getAliInstanceId()
	if instanceid != "" && err == nil {
		return instanceid
	}

	instanceid, _ = getAzureInstanceId()
	return instanceid
}
