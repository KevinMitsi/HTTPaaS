package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Instance struct {
	Host    string `json:"host"`
	IP      string `json:"ip"`
	Domain  string `json:"domain"`
	Created string `json:"created"`
}

type Store struct {
	path      string
	mu        sync.RWMutex
	instances []Instance
}

func New(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) List() []Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Instance, len(s.instances))
	copy(out, s.instances)
	return out
}

func (s *Store) Find(host string) (Instance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, inst := range s.instances {
		if inst.Host == host {
			return inst, true
		}
	}
	return Instance{}, false
}

func (s *Store) Exists(host string) bool {
	_, ok := s.Find(host)
	return ok
}

func (s *Store) Add(inst Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.instances {
		if existing.Host == inst.Host {
			return fmt.Errorf("instancia ya existe: %s", inst.Host)
		}
	}
	s.instances = append(s.instances, inst)
	return s.saveLocked()
}

func (s *Store) Delete(host string) (Instance, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, inst := range s.instances {
		if inst.Host == host {
			s.instances = append(s.instances[:i], s.instances[i+1:]...)
			return inst, true, s.saveLocked()
		}
	}
	return Instance{}, false, nil
}

func (s *Store) NextIP(base string, startOctet int) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	used := map[int]bool{}
	for _, inst := range s.instances {
		if octet, ok := extractOctet(inst.IP, base); ok {
			used[octet] = true
		}
	}

	for i := startOctet; i <= 254; i++ {
		if !used[i] {
			return fmt.Sprintf("%s%d", base, i), nil
		}
	}

	return "", errors.New("no hay IPs disponibles")
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.instances = []Instance{}
			return nil
		}
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	if err := dec.Decode(&s.instances); err != nil {
		return err
	}
	if s.instances == nil {
		s.instances = []Instance{}
	}
	return nil
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	data, err := json.MarshalIndent(s.instances, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, s.path)
}

func extractOctet(ipValue, base string) (int, bool) {
	addr := strings.TrimSpace(ipValue)
	if addr == "" {
		return 0, false
	}

	if strings.Contains(addr, "/") {
		parts := strings.Split(addr, "/")
		addr = strings.TrimSpace(parts[0])
	}

	if !strings.HasPrefix(addr, base) {
		return 0, false
	}

	raw := strings.TrimPrefix(addr, base)
	octet, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}

	return octet, true
}
