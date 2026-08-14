package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

func ComputeNextCycle(plannedExecutionTime time.Time, interval int, frequency string, days []interface{}) time.Time {
	var nextExecutionTime time.Time
	var dayInterval int
	switch frequency {
	case "minute":
		log.Println("Minute")
		nextExecutionTime = plannedExecutionTime.Add(time.Minute * time.Duration(interval))
	case "hour":
		log.Println("Hour")
		nextExecutionTime = plannedExecutionTime.Add(time.Hour * time.Duration(interval))
	case "day":
		log.Println("Day")
		nextExecutionTime = plannedExecutionTime.AddDate(0, 0, interval)
	case "week":
		log.Println("Week")
		today := int(time.Now().Weekday())
		log.Println("Today:", today)
		for index, value := range days {
			day := int(value.(float64))
			log.Print("Index in array:", index, "value in array:", day)
			if len(days) == 1 {
				dayInterval = (int((7 + day) % 7))
				nextExecutionTime = plannedExecutionTime.AddDate(0, 0, dayInterval)
			} else if len(days) > 1 {
				if index == 0 {
					dayInterval = (7 - (today - day))
					nextExecutionTime = plannedExecutionTime.AddDate(0, 0, dayInterval)
				}
				if day <= today {
					continue
				}
				if day > today {
					dayInterval = (day - today)
					nextExecutionTime = plannedExecutionTime.AddDate(0, 0, dayInterval)
					continue
				}

			} else {
				log.Println("No days configured")
			}
		}

	case "month":
		log.Println("Month")
		nextExecutionTime = plannedExecutionTime.AddDate(0, interval, 0)
	}
	return nextExecutionTime
}

func TriggerBot(tenantUniqueName string, environmentUniqueName string, botMode string, botId string, priority int, triggeredBy string, triggerMode string, plannedExecutionTime string) (int, error) {

	triggerBotUrl := os.Getenv("triggerBotUrl")

	if triggerBotUrl == "" {
		log.Println("triggerBotUrl was not configured, exiting now.")
		return 404, errors.New("triggerBotUrl was not configured")
	}

	triggerBotRequest := triggerBot{
		TenantUniqueName:      tenantUniqueName,
		EnvironmentUniqueName: environmentUniqueName,
		BotMode:               botMode,
		BotId:                 botId,
		Priority:              priority,
		TriggeredBy:           triggeredBy,
		TriggerMode:           triggerMode,
		PlannedExecutionTime:  plannedExecutionTime,
	}

	triggerBotRequestJson, _ := json.Marshal(triggerBotRequest)
	triggerBotRequestJsonByte := []byte(triggerBotRequestJson)

	client := &http.Client{}

	req, err := http.NewRequest("POST", triggerBotUrl, bytes.NewBuffer(triggerBotRequestJsonByte))
	if err != nil {
		log.Println("NewRequest: ", err)
		return req.Response.StatusCode, errors.New("HTTP Post failed")
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		log.Println("NewRequest: ", err)
		return response.StatusCode, errors.New("HTTP Post failed")
	}
	defer response.Body.Close()
	return response.StatusCode, nil
}
