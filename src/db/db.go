package db

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"log"
	"os"
	"path"
)

var localDB *leveldb.DB

var LocalDB *leveldb.DB

func StartDB(id string) error {
	exepath, err := os.Executable()
	if err != nil {
		log.Fatalf("[DB] Cannot find absolute path of executable")
		return err
	}

	homepath := path.Dir(exepath)
	localDB, err = leveldb.OpenFile(homepath+"/etc/DBFile/"+id, nil)
	LocalDB = localDB
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func clearDB() {
	batch := new(leveldb.Batch)
	iter := localDB.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		batch.Delete(key)
	}
	err := localDB.Write(batch, nil)
	if err != nil {
		log.Fatalf("clear database failed: %v", err)
	}
	iter.Release()
}

func CloseDB() {
	// clear the database
	clearDB()
	// close the database
	err := localDB.Close()
	if err != nil {
		log.Println("Error closing the database: %v", err)
	}
}

func WriteDB(key string, value DBValue) error {
	var valueSer, err = value.Serialize()
	if err != nil {
		return err
	}
	wo := &opt.WriteOptions{
		Sync: true,
	}
	err = localDB.Put([]byte(key), valueSer, wo)
	if err != nil {
		return err
	}
	return nil
}

func ReadDB(key string, value DBValue) error {
	valueSer, err := localDB.Get([]byte(key), nil)
	if err != nil {
		return err
	}
	err = value.Deserialize(valueSer)
	if err != nil {
		return err
	}
	return nil
}
