package server

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Repository is the single-writer persistence boundary. Implementations must
// replace complete documents atomically; the server owns synchronization.
type Repository interface {
	Load() (Accounts, []*Room, error)
	SaveAccounts(Accounts) error
	SaveRoom(*Room) error
}

type JSONRepository struct{ Dir string }

func atomicJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".save-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}
func (r JSONRepository) Load() (Accounts, []*Room, error) {
	a := Accounts{Users: map[string]*User{}, Sessions: map[string]Session{}, Clubs: map[string]*Club{}}
	b, err := os.ReadFile(filepath.Join(r.Dir, "accounts.json"))
	if err == nil {
		if err = json.Unmarshal(b, &a); err != nil {
			return a, nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return a, nil, err
	}
	if a.Users == nil {
		a.Users = map[string]*User{}
	}
	if a.Sessions == nil {
		a.Sessions = map[string]Session{}
	}
	if a.Clubs == nil {
		a.Clubs = map[string]*Club{}
	}
	entries, err := os.ReadDir(filepath.Join(r.Dir, "rooms"))
	if errors.Is(err, os.ErrNotExist) {
		return a, nil, nil
	}
	if err != nil {
		return a, nil, err
	}
	var rooms []*Room
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(r.Dir, "rooms", entry.Name()))
		if err != nil {
			return a, nil, err
		}
		var room Room
		if err = json.Unmarshal(b, &room); err != nil {
			return a, nil, err
		}
		rooms = append(rooms, &room)
	}
	return a, rooms, nil
}
func (r JSONRepository) SaveAccounts(a Accounts) error {
	return atomicJSON(filepath.Join(r.Dir, "accounts.json"), a)
}
func (r JSONRepository) SaveRoom(room *Room) error {
	return atomicJSON(filepath.Join(r.Dir, "rooms", room.ID+".json"), room)
}
