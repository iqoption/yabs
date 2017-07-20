package format

import (
	"io/ioutil"
	"encoding/json"
	"strconv"
	log "github.com/sirupsen/logrus"
)

type GPUInfo struct {
	Vendor   string `json:"vendor"`
	Renderer string `json:"renderer"`
}

type Info struct {
	Version  string `json:"version"`
	Browser  string `json:"browser"`
	Gpu      GPUInfo `json:"gpu"`
	Platform string `json:"platform"`
	Cpu      string `json:"cpu"`
	Ram      string `json:"ram"`
	UserId   string `json:"userid"`
}

func InfoFromFile(path string) (info *Info, err error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.WithError(err).Error("Can't read info file")
		return nil, err
	}
	var i Info
	err = json.Unmarshal(file, &i)
	if err != nil {
		log.WithError(err).Error("Can't parse info file")
		return nil, err
	}
	return &i, nil
}

func (i *Info) GetUserId() uint64 {
	var userId uint64 = 0

	if (len(i.UserId) != 0) && (i.UserId != "-1") {
		var err error = nil
		userId, err = strconv.ParseUint(i.UserId, 10, 64)
		if err != nil {
			log.WithError(err).Error("Can't convert user id to uint64")
		}
	}
	return userId
}
