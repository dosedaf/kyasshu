package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/dosedaf/kyasshu/resp"
)

const SHARD_TOTAL = 5

var ErrKeyDoesNotExist = errors.New("key does not exist")
var ErrKeyExpired = errors.New("key is expired")
var ErrCommandNotFound = errors.New("commmand not found")

var aofMtx sync.Mutex

type KeyValue struct {
	value     string
	expiresAt time.Time
}

type Shard struct {
	data map[string]KeyValue
	mtx  sync.RWMutex
}

type KeyValueStore []*Shard

func fnvHash(data string) uint32 {
	const prime = 16777619
	hash := uint32(2166136261)

	for i := 0; i < len(data); i++ {
		hash ^= uint32(data[i])
		hash *= prime
	}

	return hash
}

func NewKeyValueStore(numShards int) KeyValueStore {
	shards := make([]*Shard, numShards)

	for i := 0; i < numShards; i++ {
		//kv.data = make(map[string]KeyValue)
		shards[i] = &Shard{data: make(map[string]KeyValue)}
	}

	return shards
}

func (kv KeyValueStore) GetShardIndex(key string) int {
	hash := fnvHash(key)

	return int(hash) % len(kv)
}

func (kv KeyValueStore) Set(key string, value string) {
	shardIndex := kv.GetShardIndex(key)
	kv[shardIndex].mtx.Lock()
	kv[shardIndex].data[key] = KeyValue{value: value}
	kv[shardIndex].mtx.Unlock()
}

func (kv KeyValueStore) Get(key string) (string, error) {
	shardIndex := kv.GetShardIndex(key)
	kv[shardIndex].mtx.RLock()

	val, ok := kv[shardIndex].data[key]

	kv[shardIndex].mtx.RUnlock()

	if !ok {
		return "", ErrKeyDoesNotExist
	} else {
		if !val.expiresAt.IsZero() && !time.Now().Before(val.expiresAt) {
			kv[shardIndex].mtx.Lock()
			delete(kv[shardIndex].data, key)
			kv[shardIndex].mtx.Unlock()

			return "", ErrKeyExpired
		} else {
			//resp := fmt.Sprintf("$%d\r\n%s\r\n", len(val.value), val.value)
			return val.value, nil
		}
	}
}

func (kv KeyValueStore) Expire(key string, sec string) error {
	shardIndex := kv.GetShardIndex(key)
	kv[shardIndex].mtx.Lock()
	defer kv[shardIndex].mtx.Unlock()
	val, ok := kv[shardIndex].data[key]

	if !ok {
		return ErrKeyDoesNotExist
	} else {
		sec, err := strconv.Atoi(sec)
		if err != nil {
			return err
		} else {
			timein := time.Now().Local().Add(time.Second * time.Duration(sec))

			kv[shardIndex].data[key] = KeyValue{
				value:     val.value,
				expiresAt: timein,
			}

			return nil
		}
	}
}

func (kv KeyValueStore) TTL(key string) (int, error) {
	shardIndex := kv.GetShardIndex(key)
	kv[shardIndex].mtx.RLock()

	val, ok := kv[shardIndex].data[key]

	kv[shardIndex].mtx.RUnlock()

	if !ok {
		return -2, ErrKeyDoesNotExist
	} else {
		if val.expiresAt.IsZero() { // how do i differ this from other error
			return -1, nil
		} else if !time.Now().Before(val.expiresAt) {
			return -2, ErrKeyExpired
		} else {
			remainingSeconds := time.Until(val.expiresAt).Seconds()
			//resp := fmt.Sprintf(":%d\r\n", int(remainingSeconds))
			return int(remainingSeconds), nil
		}
	}
}

// complicated
func (kv KeyValueStore) Delete(keys []string) int {
	//var n map[shardIndex][{key, key}]string
	var deleted int
	var n [SHARD_TOTAL][]string

	for _, key := range keys {
		shardIndex := kv.GetShardIndex(key)
		n[shardIndex] = append(n[shardIndex], key)
	}

	for shardIndex, keys := range n {
		kv[shardIndex].mtx.Lock()

		for _, key := range keys {
			_, ok := kv[shardIndex].data[key]
			if ok {
				delete(kv[shardIndex].data, key)
				deleted++
			}
		}

		kv[shardIndex].mtx.Unlock()
	}

	return deleted
	/*
		var n map[int][]string
		for _, key := range keys {
			shardIndex := kv.GetShardIndex(key)
			n[shardIndex] = append(n[shardIndex], key)
		}

		for shardIndex, keys := range n {
			kv[shardIndex].mtx.Lock()
			for _, key := range keys {
				_, ok := kv[int(shardIndex)].data(key)
				if ok {
					delete(kv[k].data, key)
					deleted++
				}
			}
			kv[k].mtx.Unlock()

		}
	*/

	/*p
	var deleted int

	for _, key := range keys {
		shardIndex := kv.GetShardIndex(key)

		kv[shardIndex].mtx.Lock()
		_, ok := kv[shardIndex].data[key]
		if ok {
			delete(kv[shardIndex].data, key)
			deleted++
		}

		kv[shardIndex].mtx.Unlock()
	}

	return deleted
	*/
}

func main() {
	l, err := net.Listen("tcp4", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	// open
	f, err := os.OpenFile("aof.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Print(err)
		return
	}
	defer f.Close()

	kv := NewKeyValueStore(SHARD_TOTAL)

	fileInfo, err := os.Stat("aof.txt")
	if err != nil {
		log.Println(err)
		return
	}

	if fileInfo.Size() != 0 {
		t, err := os.Open("aof.txt")
		if err != nil {
			log.Fatal("could not open AOF file:", err)
		}
		defer t.Close()

		reader := bufio.NewReader(t)

		for {
			cmd, err := resp.Parse(reader)

			if err != nil {
				if err == io.EOF {
					break
				}

				log.Fatal("error loading from AOF:", err)
			}

			executeCommandFromAOF(kv, cmd)
		}

		log.Println("Kyasshu loaded data from AOF file.")
	}

	for {
		c, err := l.Accept()
		if err != nil {
			log.Print(err)
			return
		}

		go handleConnection(c, kv, f)
	}
}

func handleConnection(c net.Conn, kv KeyValueStore, f *os.File) {
	reader := bufio.NewReader(c)
	defer c.Close()

	for {
		cmd, err := resp.Parse(reader)
		if err != nil {
			log.Print(err)
			return
		}

		executeCommand(c, kv, cmd, f)
		fmt.Println(cmd)
	}
}

func WriteToFile(f *os.File, cmd []string) error {
	cmdSerialized := resp.SerializeCommand(cmd)
	_, err := f.Write(cmdSerialized)
	if err != nil {
		return err
	}

	return nil
}

func executeCommand(c net.Conn, kv KeyValueStore, cmd []string, f *os.File) {
	switch cmd[0] {
	case "PING":
		resp.WritePONG(c)
	case "SET":
		kv.Set(cmd[1], cmd[2])

		aofMtx.Lock()
		err := WriteToFile(f, cmd)
		if err != nil {
			log.Print(err)
		}
		aofMtx.Unlock()

		resp.WriteOK(c)
	case "GET":
		val, err := kv.Get(cmd[1])
		if err != nil {
			if errors.Is(err, ErrKeyDoesNotExist) {
				resp.WriteNullBulkString(c)
			} else if errors.Is(err, ErrKeyExpired) {
				resp.WriteNullBulkString(c)
			} else { // i don't think there are other errors but okay
				resp.WriteNullBulkString(c)
			}
		} else {
			resp.WriteBulkString(c, val)
		}

	case "EXPIRE":
		err := kv.Expire(cmd[1], cmd[2])
		if err != nil {
			// doesnt matter what's the error, we do this
			resp.WriteInteger(c, 0)
		} else {
			aofMtx.Lock()
			err := WriteToFile(f, cmd)
			if err != nil {
				log.Print(err)
			}
			aofMtx.Unlock()

			resp.WriteInteger(c, 1)
		}

	case "TTL":
		remainingSeconds, err := kv.TTL(cmd[1])
		if err != nil {
			if errors.Is(err, ErrKeyDoesNotExist) {
				resp.WriteInteger(c, -2)
			} else if errors.Is(err, ErrKeyExpired) {
				resp.WriteInteger(c, -2)
			} else {
				resp.WriteInteger(c, -2)
			}
		} else {
			// no expiration
			if remainingSeconds == -1 {
				resp.WriteInteger(c, -1)
			} else {
				resp.WriteInteger(c, int(remainingSeconds))
			}
		}

	case "DEL":
		keys := slices.Clone(cmd[1:])

		deleted := kv.Delete(keys)

		aofMtx.Lock()
		err := WriteToFile(f, cmd)
		if err != nil {
			log.Print(err)
		}
		aofMtx.Unlock()

		resp.WriteInteger(c, deleted)
	default:
		resp.WriteERR(c, "unknown command")
	}

}

func executeCommandFromAOF(kv KeyValueStore, cmd []string) error {
	switch cmd[0] {
	case "SET":
		kv.Set(cmd[1], cmd[2])

		return nil
	case "EXPIRE":
		err := kv.Expire(cmd[1], cmd[2])
		if err != nil {
			return err
		} else {
			return nil
		}

	case "DEL":
		keys := slices.Clone(cmd[1:])

		_ = kv.Delete(keys)

		return nil

	default:
		return ErrCommandNotFound
	}
}
