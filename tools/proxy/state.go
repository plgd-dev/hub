package main

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type TunnelState struct {
	SelectedTarget string `json:"selectedTarget" yaml:"selectedTarget"`
}

type State struct {
	cfg  DirectoryConfig
	data map[string]TunnelState // listener address -> state
	mux  sync.Mutex
}

func NewState(cfg DirectoryConfig) (*State, error) {
	s := &State{
		data: make(map[string]TunnelState),
		cfg:  cfg,
	}
	err := s.load()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *State) Get(id string) (TunnelState, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()

	state, ok := s.data[id]
	return state, ok
}

func (s *State) Set(id string, state TunnelState) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.data[id] = state
}

func (s *State) getFilePath() string {
	return s.cfg.Path + string(filepath.Separator) + "state.yaml"
}

func (s *State) load() error {
	f, err := os.Open(s.getFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
	}
	defer f.Close()
	return yaml.NewDecoder(f).Decode(&s.data)
}

func (s *State) dumpToFile() error {
	f, err := os.OpenFile(filepath.Clean(s.getFilePath()+".tmp"), os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	err = yaml.NewEncoder(f).Encode(s.data)
	if err != nil {
		return err
	}
	return nil
}

func (s *State) Dump() error {
	s.mux.Lock()
	defer s.mux.Unlock()
	tmpFile := filepath.Clean(s.getFilePath() + ".tmp")
	err := s.dumpToFile(tmpFile)
	if err != nil {
		return err
	}
	return os.Rename(tmpFile, s.getFilePath())
}
