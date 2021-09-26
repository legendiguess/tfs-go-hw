package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

func getPathFromFlag() string {
	pathFromFlag := flag.String("file", "", "Path to .json file with operations")

	flag.Parse()

	return *pathFromFlag
}

func getPathFromEnv() string {
	return os.Getenv("FILE")
}

func getPathFromStdin() string {
	fmt.Println("Enter path to .json file with operations:")
	var path string
	fmt.Scan(&path)
	return path
}

func getFileData() ([]byte, error) {
	inputs := []func() string{getPathFromFlag, getPathFromEnv, getPathFromStdin}

	currentInputIndex := 0

	for {
		file, err := os.Open(inputs[currentInputIndex]())
		if err != nil {
			currentInputIndex++
			if currentInputIndex == len(inputs) {
				return nil, errors.New("none of inputs contains legit file path")
			}
			continue
		}
		data, err := ioutil.ReadAll(file)
		file.Close()
		return data, err
	}
}

type OperationType int

const (
	Income = iota
	Outcome
)

type UnmarshalledOperation struct {
	Company   string
	Type      string
	Value     interface{}
	ID        interface{}
	CreatedAt string                 `json:"created_at"`
	InnerBody map[string]interface{} `json:"operation"`
}

func (unmarshalledOperation UnmarshalledOperation) getFromInnerBody(fieldName string) interface{} {
	if len(unmarshalledOperation.InnerBody) > 0 {
		if val, ok := unmarshalledOperation.InnerBody[fieldName]; ok {
			return val
		}
	}
	return nil
}

type Operation struct {
	Company   string
	Type      OperationType
	Value     int
	ID        interface{}
	CreatedAt time.Time
}

type InvalidOperation struct {
	Company string
	ID      interface{}
}

func unmarshalledToValid(unmarshalledOperations []UnmarshalledOperation) ([]Operation, []InvalidOperation) {
	var operations []Operation
	var notValidOperationsIds []InvalidOperation

	for _, unmarshalled := range unmarshalledOperations {
		var operation = Operation{}

		if unmarshalled.Company != "" {
			operation.Company = unmarshalled.Company
		} else {
			continue
		}

		var uncheckedID interface{} = unmarshalled.ID
		if uncheckedID == nil {
			uncheckedID = unmarshalled.getFromInnerBody("id")
		}

		if uncheckedID != nil {
			id, ok := uncheckedID.(float64)
			if !ok {
				id, ok := uncheckedID.(string)
				if !ok {
					continue
				} else {
					operation.ID = id
				}
			} else {
				if id == float64(int(id)) {
					operation.ID = int(id)
				} else {
					continue
				}
			}
		} else {
			continue
		}

		var uncheckedType = unmarshalled.Type

		if uncheckedType == "" {
			var myType = unmarshalled.getFromInnerBody("type")
			if myType != nil {
				uncheckedType = myType.(string)
			}
		}

		switch uncheckedType {
		case "income", "outcome":
			operation.Type = Income
		case "+", "-":
			operation.Type = Outcome
		default:
			notValidOperationsIds = append(notValidOperationsIds, InvalidOperation{Company: operation.Company, ID: operation.ID})
			continue
		}

		var uncheckedValue = unmarshalled.Value

		if uncheckedValue == nil {
			uncheckedValue = unmarshalled.getFromInnerBody("value")
		}

		if uncheckedValue != nil {
			value, ok := uncheckedValue.(float64)
			if !ok {
				value, ok := uncheckedValue.(string)
				if !ok {
					notValidOperationsIds = append(notValidOperationsIds, InvalidOperation{Company: operation.Company, ID: operation.ID})
					continue
				} else {
					v1, err := strconv.ParseFloat(value, 64)
					if err != nil {
						notValidOperationsIds = append(notValidOperationsIds, InvalidOperation{Company: operation.Company, ID: operation.ID})
						continue
					} else {
						if v1 == float64(int(v1)) {
							operation.Value = int(v1)
						} else {
							notValidOperationsIds = append(notValidOperationsIds, InvalidOperation{Company: operation.Company, ID: operation.ID})
							continue
						}
					}
				}
			} else {
				if value == float64(int(value)) {
					operation.Value = int(value)
				} else {
					notValidOperationsIds = append(notValidOperationsIds, InvalidOperation{Company: operation.Company, ID: operation.ID})
					continue
				}
			}
		} else {
			notValidOperationsIds = append(notValidOperationsIds, InvalidOperation{Company: operation.Company, ID: operation.ID})
			continue
		}

		var uncheckedCreatedAt = unmarshalled.CreatedAt

		if uncheckedCreatedAt == "" {
			var dt = unmarshalled.getFromInnerBody("created_at")
			if dt != nil {
				uncheckedCreatedAt = dt.(string)
			}
		}

		if uncheckedCreatedAt != "" {
			t, err := time.Parse(time.RFC3339, uncheckedCreatedAt)
			if err != nil {
				continue
			} else {
				operation.CreatedAt = t
			}
		} else {
			continue
		}

		operations = append(operations, operation)
	}

	return operations, notValidOperationsIds
}

type Billing struct {
	Company              string        `json:"company"`
	ValidOperationsCount int           `json:"valid_operations_count"`
	Balance              int           `json:"balance"`
	InvalidOperations    []interface{} `json:"invalid_operations,omitempty"`
}

func main() {
	data, err := getFileData()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	var unmarshalledOperations []UnmarshalledOperation

	if err := json.Unmarshal(data, &unmarshalledOperations); err != nil {
		panic(err)
	}

	operations, invalidOperations := unmarshalledToValid(unmarshalledOperations)

	var billings []Billing

	for _, operation := range operations {
		var company = operation.Company
		var billingIndex = -1
		for index, billing := range billings {
			if billing.Company == company {
				billingIndex = index
				break
			}
		}

		if billingIndex == -1 {
			billings = append(billings, Billing{})
			billingIndex = len(billings) - 1
			billings[billingIndex].Company = company
		}

		billings[billingIndex].ValidOperationsCount++
		if operation.Type == Income {
			billings[billingIndex].Balance += operation.Value
		} else {
			billings[billingIndex].Balance -= operation.Value
		}
	}

	for _, invalidOperation := range invalidOperations {
		for index, billing := range billings {
			if billing.Company == invalidOperation.Company {
				billings[index].InvalidOperations = append(billings[index].InvalidOperations, invalidOperation.ID)
				break
			}
		}
	}

	file, _ := json.MarshalIndent(billings, "", "\t")

	_ = ioutil.WriteFile("out.json", file, 0644)
}
