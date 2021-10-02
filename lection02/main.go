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
	InvalidType = OperationType(iota)
	IncomeType
	OutcomeType
)

func (opType *OperationType) UnmarshalJSON(data []byte) error {
	var input string
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	switch input {
	case "income", "+":
		*opType = IncomeType
	case "outcome", "-":
		*opType = OutcomeType
	default:
		*opType = InvalidType
	}

	return nil
}

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var input string
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	if input != "" {
		parsedTime, _ := time.Parse(time.RFC3339, input)
		t.Time = parsedTime
	}

	return nil
}

type Value int

const (
	InvalidValue = Value(0)
)

func (v *Value) UnmarshalJSON(data []byte) error {
	var input interface{}
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	if input != nil {
		value, ok := input.(float64)
		if !ok {
			str, ok := input.(string)
			if !ok {
				return nil
			}
			var err error
			value, err = strconv.ParseFloat(str, 64)
			if err != nil {
				return nil
			}
		}
		if floatIsInteger(value) {
			*v = Value(int(value))
		}
	}
	return nil
}

type InnerBody struct {
	Value     Value
	ID        interface{}
	Type      OperationType
	CreatedAt Time `json:"created_at"`
}

type RawOperation struct {
	Company   string
	Type      OperationType
	Value     Value
	ID        interface{}
	CreatedAt Time      `json:"created_at"`
	InnerBody InnerBody `json:"operation"`
}

func (rawOperation RawOperation) getValue() Value {
	if rawOperation.Value == InvalidValue {
		return rawOperation.InnerBody.Value
	}
	return rawOperation.Value
}

func (rawOperation RawOperation) getID() interface{} {
	if rawOperation.ID == nil {
		return rawOperation.InnerBody.ID
	}
	return rawOperation.ID
}

func (rawOperation RawOperation) getType() OperationType {
	if rawOperation.Type == InvalidType {
		return rawOperation.InnerBody.Type
	}
	return rawOperation.Type
}

func (rawOperation RawOperation) getCreatedAt() Time {
	if rawOperation.CreatedAt.Time.IsZero() {
		return rawOperation.InnerBody.CreatedAt
	}
	return rawOperation.CreatedAt
}

type Operation struct {
	Company string
	ID      interface{}
	InnerBody
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

	rawTime := rawOperation.getCreatedAt()
	if rawTime.Time.IsZero() {
		return nil, Skip
	}
	operation.CreatedAt = rawTime

	mType := rawOperation.getType()
	if mType == InvalidType {
		return &operation, Invalid
	}
	operation.Type = mType

	value := rawOperation.getValue()
	if value == InvalidValue {
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
		intValue := int(operation.Value)
		if operation.Type == IncomeType {
			billings[billingIndex].Balance += intValue
		} else {
			billings[billingIndex].Balance -= intValue
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
