package main

import (
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"math/rand"
	"sync"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var mux sync.RWMutex
var db *bolt.DB

//https://zupzup.org/boltdb-example/
// Loads the database; Creates one if does not exist
func LoadDB(dataBaseName string) error {
	var err error
	db, err = bolt.Open(dataBaseName, 0600, nil)
	if err != nil {
		return fmt.Errorf("could not open db, %v", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("DB"))
		if err != nil {
			return fmt.Errorf("could not create root bucket: %v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("could not set up buckets, %v", err)
	}
	log.Println("DB Setup Done")
	return nil
}

//Inserts a link into the database
//Returns the ID for the link to be shared later
func InsertValue(Value string) (string, error) {
	mux.Lock()
	defer mux.Unlock()
	for {
		key := generateRandomStringAsByte()
		hasValue := false
		//Check if the random exists in database
		err := db.View(func(tx *bolt.Tx) error {
			check := tx.Bucket([]byte("DB")).Get(key)
			hasValue = check != nil
			return nil
		})
		if err != nil {
			return "", err
		}
		if !hasValue { //In this case we save the string into database
			err = db.Update(func(tx *bolt.Tx) error {
				err = tx.Bucket([]byte("DB")).Put(key, []byte(Value))
				if err != nil {
					return fmt.Errorf("could not read db: %v", err)
				}
				return nil
			})
			if err != nil {
				return "", err
			}
			return string(key), nil
		}
	}
}

//Check if a key exists; On errors return false as well
func HasKey(Key string) bool {
	hasValue := false
	mux.RLock()
	//Check if the random exists in database
	_ = db.View(func(tx *bolt.Tx) error {
		check := tx.Bucket([]byte("DB")).Get([]byte(Key))
		hasValue = check != nil
		return nil
	})
	mux.RUnlock()
	return hasValue
}

//Remove a key from the database
func RemoveKey(Key string) error {
	if !HasKey(Key) {
		return fmt.Errorf("this token does not exits")
	}
	mux.Lock()
	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("DB")).Delete([]byte(Key))
		if err != nil {
			return fmt.Errorf("could not delete key: %v", err)
		}
		return nil
	})
	mux.Unlock()
	return err
}

//List all of the values
func ListAllValues() (map[string]string, error) {
	m := make(map[string]string)
	mux.RLock()
	err := db.View(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("DB")).ForEach(func(k, v []byte) error {
			if len(v) > 100 {
				m[string(k)] = string(v[:100]) + " *...* "
			} else {
				m[string(k)] = string(v)
			}
			return nil
		})
		return err
	})
	mux.RUnlock()
	return m, err
}

//Read the value from database
func ReadValue(Key string) (string, error) {
	var res string
	mux.RLock()
	err := db.View(func(tx *bolt.Tx) error {
		check := tx.Bucket([]byte("DB")).Get([]byte(Key))
		if check == nil {
			return fmt.Errorf("Cannot find value for " + Key)
		}
		res = string(check)
		return nil
	})
	mux.RUnlock()
	if err != nil {
		return "", err
	}
	return res, nil
}

//Keys are 8 letter long
func generateRandomStringAsByte() []byte {
	s := ""
	for i := 0; i < 8; i++ {
		s += string(alphabet[rand.Int31n(52)])
	}
	return []byte(s)
}
