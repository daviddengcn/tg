package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func loadFromJenkins(url string) (string, error) {
	resp, err := http.Get(url + "/api/json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var report struct {
		Stderr string
		Stdout string
	}

	if err := json.Unmarshal(body, &report); err != nil {
		return "", err
	}
	
	if len(report.Stdout) > len(report.Stderr)*10 {
		return report.Stdout, nil
	}

	return report.Stderr, nil
}
