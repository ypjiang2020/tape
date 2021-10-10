package chaincode

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
)

type VersionedValue struct {
	Txid string
	Val  []byte
}

type TempDB struct {
	mutex    sync.Mutex
	Sessions map[string]*SessionDB
}

type WriteSet struct {
	keys []string
	vals [][]byte
}

func (ws *WriteSet) append(key string, val []byte) {
	ws.keys = append(ws.keys, key)
	ws.vals = append(ws.vals, val)
}

type SessionDB struct {
	session   string
	db        map[string][]byte    // key is key
	writeSets map[string]*WriteSet // key is txid
}

func NewTempDB() *TempDB {
	return newTempDB()
}

func newSessionDB(session string) *SessionDB {
	return &SessionDB{
		session:   session,
		db:        map[string][]byte{},
		writeSets: map[string]*WriteSet{},
	}
}

func (sdb *SessionDB) Get(key string) *VersionedValue {
	// TODO: do we need to read from writeSets?
	val, ok := sdb.db[key]
	if !ok {
		return nil
	}

	versionedValue := &VersionedValue{}
	err := json.Unmarshal(val, versionedValue)
	if err != nil {
		log.Fatalln("get from session db,", key, versionedValue.Txid, err)
	}
	return versionedValue
}

func (sdb *SessionDB) Put(txid, key string, val []byte) {
	if _, ok := sdb.writeSets[txid]; !ok {
		sdb.writeSets[txid] = &WriteSet{}
	}
	sdb.writeSets[txid].append(key, val)
}

func (sdb *SessionDB) Rollback(txid string) {
	delete(sdb.writeSets, txid)
}

func (sdb *SessionDB) Commit(txid string) {
	if _, ok := sdb.writeSets[txid]; !ok {
		log.Fatalln("write set not found", txid)
	}
	writeset := sdb.writeSets[txid]
	cnt := len(writeset.keys)
	for i := 0; i < cnt; i++ {
		var val = writeset.vals[i]
		// verval := &VersionedValue{
		// 	Txid: txid,
		// 	Val:  writeset.vals[i],
		// }
		// val, err := json.Marshal(verval)
		// if err != nil {
		// 	log.Fatalf("commit to session db, marshal %v", err)

		// }
		sdb.db[writeset.keys[i]] = val
	}
	delete(sdb.writeSets, txid)
}

func newTempDB() *TempDB {
	res := &TempDB{
		Sessions: map[string]*SessionDB{},
	}
	return res
}

func (tdb *TempDB) Get(key, session string) *VersionedValue {
	if session == "" {
		return nil
	}
	if sdb, ok := tdb.Sessions[session]; ok {
		return sdb.Get(key)
	} else {
		return nil
	}
}

func (tdb *TempDB) Put(key, txid string, val []byte) {
	// val has already been encoded by marshaling (txid, ori_val)
	session := GetSessionFromTxid(txid)
	if session == "" {
		return
	}
	if sdb, ok := tdb.Sessions[session]; ok {
		sdb.Put(txid, key, val)
	} else {
		db := newSessionDB(session)
		db.Put(txid, key, val)
		tdb.Sessions[session] = db
	}
}

func (tdb *TempDB) Rollback(txid string) {
	session := GetSessionFromTxid(txid)
	if session == "" {
		return
	}
	if sdb, ok := tdb.Sessions[session]; ok {
		sdb.Rollback(txid)
	} else {
		log.Fatalln("something is wrong with the Temp DB, please check it")
	}
}
func (tdb *TempDB) Commit(txid string) {
	session := GetSessionFromTxid(txid)
	if session == "" {
		return
	}
	if sdb, ok := tdb.Sessions[session]; ok {
		sdb.Commit(txid)
	} else {
		log.Fatalln("something is wrong with the Temp DB, please check it")
	}
}

// Prune: delete the session that txid belongs to.
func (tdb *TempDB) Prune(txid string) {
	session := GetSessionFromTxid(txid)
	delete(tdb.Sessions, session)
}

func (tdb *TempDB) String() string {
	var res string
	for session, sessiondb := range tdb.Sessions {
		res += fmt.Sprintf("session:%s\n", session)
		res += fmt.Sprintf("\tcommitdb:\n")
		for key, val := range sessiondb.db {
			var verval VersionedValue
			err := json.Unmarshal(val, &verval)
			if err != nil {
				log.Fatalf("stringfy tempdb, unmarshal error: %v", err)

			}
			res += fmt.Sprintf("\t\tkey=%s; txid=%s, val=%s\n", key, verval.Txid, verval.Val)
		}
		res += fmt.Sprintf("\tuncommitdb:\n")
		for txid, rws := range sessiondb.writeSets {
			res += fmt.Sprintf("\t\ttxid=%s:\n", txid)
			for i := 0; i < len(rws.keys); i++ {
				res += fmt.Sprintf("\t\t\tkey=%s; val=%s\n", rws.keys[i], string(rws.vals[i]))
			}
		}
	}
	res += fmt.Sprintf("\n")
	return res
}

// txid format: seqNumber_Session_oriTxid
func GetSessionFromTxid(txid string) string {
	temp := strings.Split(txid, "_+=+_")
	if len(temp) == 1 {
		return ""
	} else if len(temp) == 3 {
		return temp[1]
	}
	return ""
}

func GetSeqFromTxid(txid string) string {
	temp := strings.Split(txid, "_+=+_")
	if len(temp) == 1 {
		return ""
	} else if len(temp) == 3 {
		return temp[0]
	}
	return ""
}
