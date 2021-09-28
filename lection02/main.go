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

func getDataFromFilePath(filepath string) ([]byte, bool) {
	file, err := os.Open(filepath)

	if err != nil {
		return nil, false
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, false
	}

	file.Close()
	return data, true
}

func getDataFromInputs() ([]byte, error) {
	data, ok := getDataFromFilePath(getPathFromFlag())

	if ok {
		return data, nil
	}

	data, ok = getDataFromFilePath(getPathFromEnv())

	if ok {
		return data, nil
	}

	data, err := ioutil.ReadAll(os.Stdin)

	if err == nil {
		return data, nil
	}

	return nil, errors.New("none of the inputs are correct")
}

type OperationType int

const (
	Income = iota
	Outcome
)

type RawOperation struct {
	Company   string
	Type      string
	Value     interface{}
	ID        interface{}
	CreatedAt string                 `json:"created_at"`
	InnerBody map[string]interface{} `json:"operation"`
}

func (rawOperation RawOperation) getFromInnerBody(fieldName string) interface{} {
	if len(rawOperation.InnerBody) > 0 {
		if val, ok := rawOperation.InnerBody[fieldName]; ok {
			return val
		}
	}
	return nil
}

func (rawOperation RawOperation) getValue() interface{} {
	if rawOperation.Value == nil {
		return rawOperation.getFromInnerBody("value")
	}
	return rawOperation.Value
}

func (rawOperation RawOperation) getID() interface{} {
	if rawOperation.ID == nil {
		return rawOperation.getFromInnerBody("id")
	}
	return rawOperation.ID
}

func (rawOperation RawOperation) getType() string {
	if rawOperation.Type == "" {
		var typeFromInner = rawOperation.getFromInnerBody("type")
		if typeFromInner != nil {
			return typeFromInner.(string)
		}
	}
	return rawOperation.Type
}

func (rawOperation RawOperation) getCreatedAt() string {
	if rawOperation.CreatedAt == "" {
		var createdAt = rawOperation.getFromInnerBody("created_at")
		if createdAt != nil {
			return createdAt.(string)
		}
	}
	return rawOperation.CreatedAt
}

type Operation struct {
	Company   string
	Type      OperationType
	Value     int
	ID        interface{}
	CreatedAt time.Time
}

type OperationStatus int

const (
	Valid = iota
	Invalid
	Skip
)

func floatIsInteger(value float64) bool {
	return value == float64(int(value))
}

func rawIDToValid(value interface{}) (interface{}, bool) {
	if value != nil {
		id, ok := value.(float64)
		if !ok {
			id, ok := value.(string)
			if !ok {
				return nil, false
			}
			return id, true
		}
		if !floatIsInteger(id) {
			return nil, false
		}
		return int(id), true
	}
	return nil, false
}

func rawTimeToValid(input string) (time.Time, bool) {
	if input != "" {
		t, err := time.Parse(time.RFC3339, input)
		return t, err == nil
	}
	return time.Time{}, false
}

func rawValueToValid(input interface{}) (int, bool) {
	if input != nil {
		value, ok := input.(float64)
		if !ok {
			str, ok := input.(string)
			if !ok {
				return 0, false
			}
			var err error
			value, err = strconv.ParseFloat(str, 64)
			if err != nil {
				return 0, false
			}
		}
		return int(value), floatIsInteger(value)
	}
	return 0, false
}

func rawTypeToValid(input string) (OperationType, bool) {
	switch input {
	case "income", "+":
		return Income, true
	case "outcome", "-":
		return Outcome, true
	}
	return OperationType(-1), false
}

func newOperationFromRaw(rawOperation RawOperation) (*Operation, OperationStatus) {
	var operation = Operation{}

	if rawOperation.Company != "" {
		operation.Company = rawOperation.Company
	} else {
		return nil, Skip
	}

	id, ok := rawIDToValid(rawOperation.getID())
	if !ok {
		return nil, Skip
	}
	operation.ID = id

	time, ok := rawTimeToValid(rawOperation.getCreatedAt())
	if !ok {
		return nil, Skip
	}
	operation.CreatedAt = time

	mType, ok := rawTypeToValid(rawOperation.getType())
	if !ok {
		return &operation, Invalid
	}
	operation.Type = mType

	value, ok := rawValueToValid(rawOperation.getValue())
	if !ok {
		return &operation, Invalid
	}
	operation.Value = value

	return &operation, Valid
}

type InvalidOperation struct {
	Company string
	ID      interface{}
}

func rawOperationsToValid(rawOperations []RawOperation) ([]Operation, []InvalidOperation) {
	var operations []Operation
	var invalidOperations []InvalidOperation

	for _, raw := range rawOperations {
		operation, status := newOperationFromRaw(raw)
		switch status {
		case Valid:
			operations = append(operations, *operation)
		case Invalid:
			invalidOperations = append(invalidOperations, InvalidOperation{Company: operation.Company, ID: operation.ID})
		}
	}
	return operations, invalidOperations
}

type Billing struct {
	Company              string        `json:"company"`
	ValidOperationsCount int           `json:"valid_operations_count"`
	Balance              int           `json:"balance"`
	InvalidOperations    []interface{} `json:"invalid_operations,omitempty"`
}

func billingsFromOperations(operations []Operation, invalidOperations []InvalidOperation) []Billing {
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

	return billings
}

func main() {
	data, err := getDataFromInputs()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	var rawOperations []RawOperation

	if err := json.Unmarshal(data, &rawOperations); err != nil {
		panic(err)
	}

	operations, invalidOperations := rawOperationsToValid(rawOperations)

	billings := billingsFromOperations(operations, invalidOperations)

	file, _ := json.MarshalIndent(billings, "", "\t")

	_ = ioutil.WriteFile("out.json", file, 0644)
}
