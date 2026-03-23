package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

var ErrCASConflict = errors.New("snapshot revision conflict")

type Metadata struct {
	SchemaVersion string `json:"schemaVersion"`
	Generator     string `json:"generator"`
	GeneratedAt   string `json:"generatedAt"`
	Revision      int64  `json:"revision"`
}

func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func LoadJSON(path string, target any) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, target)
}

func LoadJSONIfExists(path string, target any) (bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, json.Unmarshal(payload, target)
}

func CurrentRevision(path string) (int64, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	var head struct {
		Revision int64 `json:"revision"`
	}
	if err := json.Unmarshal(payload, &head); err != nil {
		return 0, err
	}
	return head.Revision, nil
}

func WriteSnapshot(path string, document any, generator string, expectedRevision int64) (Metadata, error) {
	currentRevision, err := CurrentRevision(path)
	if err != nil {
		return Metadata{}, err
	}
	if currentRevision != expectedRevision {
		return Metadata{}, fmt.Errorf("%w: expected %d got %d", ErrCASConflict, expectedRevision, currentRevision)
	}
	meta := Metadata{
		SchemaVersion: "1.0",
		Generator:     generator,
		GeneratedAt:   NowUTC(),
		Revision:      currentRevision + 1,
	}
	if err := setMetadata(document, meta); err != nil {
		return Metadata{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Metadata{}, err
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return Metadata{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return Metadata{}, err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(append(payload, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return Metadata{}, err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return Metadata{}, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return Metadata{}, err
	}
	return meta, nil
}

func setMetadata(document any, meta Metadata) error {
	value := reflect.ValueOf(document)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return errors.New("document must be a non-nil pointer")
	}
	target := value.Elem()
	if target.Kind() != reflect.Struct {
		return errors.New("document must point to a struct")
	}
	field := target.FieldByName("Metadata")
	if !field.IsValid() || !field.CanSet() {
		return errors.New("document is missing settable Metadata field")
	}
	if field.Type() != reflect.TypeOf(Metadata{}) {
		return errors.New("Metadata field must use state.Metadata")
	}
	field.Set(reflect.ValueOf(meta))
	return nil
}
