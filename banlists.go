package main

import (
	"strings"
	"sync"
)

// StringList - структура для хранения строк с мьютексом
type StringList struct {
	mu    sync.RWMutex
	items []string
}

// Add добавляет строку в список
func (s *StringList) Add(item string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
}

// Contains проверяет, содержится ли строка в списке
func (s *StringList) Contains(item string) bool {
	for _, str := range s.items {
		if strings.Contains(item, str) {
			return true
		}
	}
	return false
}
