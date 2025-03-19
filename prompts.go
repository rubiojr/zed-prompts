package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cloudflare/lmdb-go/lmdb"
)

// Metadata examples from metadata.v2
// Key: {"kind":"User","uuid":"<uuid>"}
//
//	Value: {
//		  ID: {
//		    Kind:User,
//		    UUID:"<uuid>"
//		  },
//		  Title:"<title>",
//		  Default:false,
//		  SavedAt:"2025-03-19T11:36:31.593182037Z"
//		}
//
// Body examples from bodies.v2
// Key: {"kind":"Body","uuid":"<uuid>"}
// Value: <raw prompt text>
type ID struct {
	Kind string `json:"kind"`
	UUID string `json:"uuid"`
}

type Metadata struct {
	ID      ID     `json:"id"`
	Title   string `json:"title"`
	Default bool   `json:"default"`
	SavedAt string `json:"saved_at"`
}

type Prompt struct {
	Metadata Metadata `json:"metadata"`
	Content  string   `json:"content"`
}

type Body struct {
	ID      ID     `json:"id"`
	Content string `json:"content"`
}

func openDB(path string) (*lmdb.Env, error) {
	if _, err := os.Stat(path); err != nil && err != os.ErrNotExist {
		return nil, fmt.Errorf("prompts database does not exist")
	}

	env, err := lmdb.NewEnv()
	if err != nil {
		return nil, err
	}

	err = env.SetMaxDBs(2)
	if err != nil {
		return nil, err
	}

	err = env.SetMapSize(1 << 30)
	if err != nil {
		return nil, err
	}

	err = env.Open(path, 0, 0644)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func getBody(txn *lmdb.Txn, uuid string) (string, error) {
	dbi, err := txn.OpenDBI("bodies.v2", 0)
	if err != nil {
		return "", fmt.Errorf("error opening bodies database: %w", err)
	}

	// Create the key in the expected format
	key := ID{
		Kind: "User",
		UUID: uuid,
	}

	keyJSON, err := json.Marshal(key)
	if err != nil {
		return "", fmt.Errorf("error creating key: %w", err)
	}

	// Get the value from the database
	val, err := txn.Get(dbi, keyJSON)
	if err != nil {
		if lmdb.IsNotFound(err) {
			return "", fmt.Errorf("body with UUID %s not found", uuid)
		}
		return "", fmt.Errorf("error getting body: %w", err)
	}

	return string(val), nil
}

func export(dbPath string, output string) error {
	env, err := openDB(dbPath)
	if err != nil {
		return err
	}
	defer env.Close()

	metadata := make([]Metadata, 0)
	err = env.View(func(txn *lmdb.Txn) error {
		metadata, err = getAllMetadata(txn)
		return err
	})

	if err != nil {
		return err
	}

	db := []*Prompt{}
	for _, m := range metadata {
		var body string
		err = env.View(func(txn *lmdb.Txn) error {
			body, err = getBody(txn, m.ID.UUID)
			return err
		})
		if err != nil {
			return err
		}

		db = append(db, &Prompt{
			Metadata: m,
			Content:  body,
		})
	}

	jsonBytes, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}

	// export to stdout requested
	if output == "-" {
		fmt.Println(string(jsonBytes))
		return nil
	}

	return os.WriteFile(output, jsonBytes, 0644)
}

func getAllMetadata(txn *lmdb.Txn) ([]Metadata, error) {
	dbi, err := txn.OpenDBI("metadata.v2", 0)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	cursor, err := txn.OpenCursor(dbi)
	if err != nil {
		return nil, fmt.Errorf("creating cursor: %w", err)
	}
	defer cursor.Close()

	metadatas := make([]Metadata, 0)
	for {
		_, value, err := cursor.Get(nil, nil, lmdb.Next)
		if lmdb.IsNotFound(err) {
			break // No more records
		}
		if err != nil {
			return nil, fmt.Errorf("iterating database: %w", err)
		}

		var metadata Metadata
		if err := json.Unmarshal(value, &metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling metadata: %w", err)
		}

		metadatas = append(metadatas, metadata)
	}

	return metadatas, nil
}

func getAllBodies(txn *lmdb.Txn) ([]Body, error) {
	dbi, err := txn.OpenDBI("bodies.v2", 0)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	cursor, err := txn.OpenCursor(dbi)
	if err != nil {
		return nil, fmt.Errorf("creating cursor: %w", err)
	}
	defer cursor.Close()

	bodies := make([]Body, 0)
	for {
		key, value, err := cursor.Get(nil, nil, lmdb.Next)
		if lmdb.IsNotFound(err) {
			break // No more records
		}
		if err != nil {
			return nil, fmt.Errorf("iterating database: %w", err)
		}

		var body Body
		if err := json.Unmarshal(key, &body.ID); err != nil {
			return nil, fmt.Errorf("unmarshaling body: %w", err)
		}
		body.Content = string(value)
		bodies = append(bodies, body)
	}

	return bodies, nil
}

func put(txn *lmdb.Txn, database string, key []byte, value []byte) error {
	dbi, err := txn.OpenDBI(database, lmdb.Create)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}

	if err := txn.Put(dbi, []byte(key), value, 0); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}

	return nil
}

func dbPath() string {
	if runtime.GOOS == "linux" {
		return filepath.Join(os.Getenv("HOME"), ".local/share/zed/prompts/prompts-library-db.0.mdb")
	} else if runtime.GOOS == "darwin" {
		return filepath.Join(os.Getenv("HOME"), ".config/zed/prompts")
	}
	panic("unsupported OS")
}

func listMetadata() error {
	env, err := openDB(dbPath())
	if err != nil {
		return err
	}
	defer env.Close()

	err = env.View(func(txn *lmdb.Txn) error {
		dbi, err := txn.OpenDBI("metadata.v2", 0)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}

		cursor, err := txn.OpenCursor(dbi)
		if err != nil {
			return fmt.Errorf("creating cursor: %w", err)
		}
		defer cursor.Close()

		for {
			key, _, err := cursor.Get(nil, nil, lmdb.Next)
			if lmdb.IsNotFound(err) {
				break // No more records
			}
			if err != nil {
				return fmt.Errorf("iterating database: %w", err)
			}

			//fmt.Printf("Key: %s, Value: %s\n", key, value)
			fmt.Println(string(key))
		}
		return nil
	})

	return nil
}

func importJSON(input string, dbPath string) error {
	env, err := openDB(dbPath)
	if err != nil {
		return err
	}
	defer env.Close()

	db := []*Prompt{}

	var jsonBytes []byte
	if input == "-" {
		// read from stdin
		jsonBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
	} else {
		jsonBytes, err = os.ReadFile(input)
		if err != nil {
			return err
		}
	}
	if err := json.Unmarshal(jsonBytes, &db); err != nil {
		return err
	}

	err = env.Update(func(txn *lmdb.Txn) (err error) {
		for _, prompt := range db {
			mjson, err := json.Marshal(prompt.Metadata)
			if err != nil {
				return err
			}
			idjson, err := json.Marshal(prompt.Metadata.ID)
			if err != nil {
				return err
			}

			err = put(txn, "metadata.v2", []byte(idjson), mjson)
			if err != nil {
				return err
			}
			err = put(txn, "bodies.v2", []byte(idjson), []byte(prompt.Content))
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}
