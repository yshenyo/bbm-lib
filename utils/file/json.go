package file

import (
	"encoding/json"
	"io/ioutil"
)

func WriteDebugJsonFile(data interface{}, file string, format bool) error {
	if format {
		return WriteFormatJsonFile(data, file)
	}
	return WriteMiniJsonFile(data, file)
}

func WriteMiniJsonFile(data interface{}, file string) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, b, 0777)
	if err != nil {
		return err
	}
	return nil
}

func WriteFormatJsonFile(data interface{}, file string) error {
	b, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, b, 0777)
	if err != nil {
		return err
	}
	return nil
}
