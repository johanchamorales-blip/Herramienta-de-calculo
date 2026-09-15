package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type JSONStore[T any] struct {
	Path string
	Mu   sync.Mutex
}

func (s *JSONStore[T]) Read() ([]T, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return s.readUnlocked()
}

func (s *JSONStore[T]) Write(items []T) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return s.writeUnlocked(items)
}

func (s *JSONStore[T]) readUnlocked() ([]T, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, fmt.Errorf("leer %s: %w", s.Path, err)
	}
	var items []T
	if len(data) == 0 {
		return []T{}, nil
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parsear %s: %w", s.Path, err)
	}
	return items, nil
}

func (s *JSONStore[T]) writeUnlocked(items []T) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
