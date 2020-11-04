/*

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package redis

import (
	"fmt"
	"log"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis"
)

// Client stores essential client info
type Client struct {
	DefaultDB *redis.Client
	DBOne     *redis.Client
}

const (
	redisSocketAddr string = "/var/run/redis/redis.sock"
	// RSUCCESS is the success code returned by a Redis op
	RSUCCESS int64 = 1
	// RFAIL is the failure code returned by a Redis op
	RFAIL int64 = 0
)

// TODO: Currently we have host/path --> ip --> port
// if ports are the same for host/path, then it might make more sense,
// in short terms, to have host/path --> ip, host/path --> port instead

// Init initializes the redis clients
func Init() (*Client, error) {
	rClient, err := CreateRedisClient() // connecting to redis
	if err != nil {
		return nil, fmt.Errorf("Failed connecting to Redis: %s", err.Error())
	}
	err = rClient.Flush() // when the program starts, flush all stale memory
	if err != nil {
		return nil, fmt.Errorf("Failed to FlushAll: %s", err.Error())
	}
	return rClient, nil
}

func InitForTesting() (*Client, error) {
	mr, err := miniredis.Run()

	if err != nil {
		return nil, err
	}

	defaultDB := redis.NewClient(&redis.Options{
		Addr: mr.Addr(), // connect to domain socket
		DB:   0,         // use default DB
	})

	_, err = defaultDB.Ping().Result()
	if err != nil {
		return nil, err
	}

	dbOne := redis.NewClient(&redis.Options{
		Addr: mr.Addr(), // connect to domain socket
		DB:   1,         // use DB number 1
	})

	_, err = dbOne.Ping().Result()
	if err != nil {
		return nil, err
	}

	return &Client{defaultDB, dbOne}, nil

}

// CreateRedisClient establishes connection to redis DB
func CreateRedisClient() (*Client, error) {
	defaultDB := redis.NewClient(&redis.Options{
		Network:  "unix",          // use default Addr
		Addr:     redisSocketAddr, // connect to domain socket
		Password: "",              // no password set
		DB:       0,               // use default DB
	})

	_, err := defaultDB.Ping().Result()
	if err != nil {
		return nil, err
	}

	dbOne := redis.NewClient(&redis.Options{
		Network:  "unix",          // use default Addr
		Addr:     redisSocketAddr, // connect to domain socket
		Password: "",              // no password set
		DB:       1,               // use DB number 1
	})

	_, err = dbOne.Ping().Result()
	if err != nil {
		return nil, err
	}

	return &Client{defaultDB, dbOne}, nil
}

//--------------------- Default DB: svc port --> []IPport ------------------------

// DefaulDBSAdd does SAdd on Default DB and logs results
func (c *Client) DefaultDBSAdd(svcport, ipport string) {
	_, err := c.DefaultDB.SAdd(svcport, ipport).Result()
	if err != nil {
		log.Printf("DefaultDB.SAdd(%s, %s).Result() Error: %s\n", svcport, ipport, err.Error())
	}
}

// DefaultDBDel does Del on Default DB and logs results
func (c *Client) DefaultDBDel(svcport string) {
	// then delete host from Default DB
	_, err := c.DefaultDB.Del(svcport).Result()
	if err != nil {
		log.Printf("DefaultDB.Del(%s).Result() Error: %s\n", svcport, err.Error())
	}
}

// DefaultDBSUnionStore does sunionstore on default db
func (c *Client) DefaultDBSUnionStore(dest, src string) {
	_, err := c.DefaultDB.SUnionStore(dest, src).Result()
	if err != nil {
		log.Printf("DefaultDB.SUnionStore(%s, %s).Result() Error: %s\n", dest, src, err.Error())
	}
}

//----------------------- DB One: hostport --> []svc port -------------------------------

// DBOneSAdd does SAdd on DB One and logs results
func (c *Client) DBOneSAdd(hostport, svcport string) {
	_, err := c.DBOne.SAdd(hostport, svcport).Result()

	if err != nil {
		log.Printf("DBOne.SAdd(%s, %s).Result() Error: %s\n", hostport, svcport, err.Error())
	}
}

// DBOneSRem does SRem on DB One and logs results
func (c *Client) DBOneSRem(hostport, svcport string) {
	_, err := c.DBOne.SRem(hostport, svcport).Result()
	if err != nil {
		log.Printf("DBOne.SRem(%s, %s).Result() Error: %s\n", hostport, svcport, err.Error())
	}
}

// DBOneDel does Del on Default DB and logs results
func (c *Client) DBOneDel(hostport string) {
	// then delete host from DB One
	_, err := c.DBOne.Del(hostport).Result()
	if err != nil {
		log.Printf("DBOne.Del(%s).Result() Error: %s\n", hostport, err.Error())
	}
}

// DefaultDBSUnionStore does sunionstore on default db
func (c *Client) DBOneSUnionStore(dest, src string) {
	_, err := c.DBOne.SUnionStore(dest, src).Result()
	if err != nil {
		log.Printf("DBOne.SUnionStore(%s, %s).Result() Error: %s\n", dest, src, err.Error())
	}
}

//------------------------- Other ---------------------------------------------

// Flush flushes all of redis database
func (c *Client) Flush() error {
	if _, err := c.DefaultDB.FlushAll().Result(); err != nil {
		return err
	}
	return nil
}

// Close tries to close the 2 clients
func (c *Client) Close() {
	c.DefaultDB.Close()
	c.DBOne.Close()
	// for garbage collector
	c.DefaultDB = nil
	c.DBOne = nil
}

// Terminate tries to flush the entire redis and close clients
func (c *Client) Terminate() {
	c.Flush() // should go first
	c.Close()
}

// PrintAllKeys prints all the keys in the redis client. For debugging purposes etc
func (c *Client) PrintAllKeys() {
	var (
		res interface{}
		err error
	)
	if res, err = c.DefaultDB.Do("KEYS", "*").Result(); err != nil {
		log.Println("Error Printing Default DB (0): ", err)
	} else {
		log.Println("DefaultDB.Do(\"KEYS\", \"*\").Result(): ", res)
	}

	if res, err = c.DBOne.Do("KEYS", "*").Result(); err != nil {
		log.Println("Error Printing DB One (1): ", err)
	} else {
		log.Println("DBOne.Do(\"KEYS\", \"*\").Result(): ", res)
	}
}

func (c *Client) GetDefaultDBKeyValues() map[string][]string {
	var (
		res         interface{}
		err         error
		keyValueMap map[string][]string
	)

	if res, err = c.DefaultDB.Do("KEYS", "*").Result(); err != nil {
		log.Println("Error Printing DB One (1): ", err)
	}

	keyValueMap = make(map[string][]string)

	switch keys := res.(type) {
	case []interface{}:
		for _, key := range keys {
			if smembers, err := c.DefaultDB.Do("SMEMBERS", key).Result(); err != nil {
				log.Println("Error Printing DB One (1): ", err)
			} else {
				keyValueMap[key.(string)] = []string{}
				switch values := smembers.(type) {
				case []interface{}:
					for _, value := range values {
						keyValueMap[key.(string)] = append(keyValueMap[key.(string)], value.(string))
					}
				default:
					fmt.Printf("Cannot iterate over %T\n", smembers)
				}
			}
		}
	default:
		fmt.Printf("Cannot iterate over %T\n", res)
	}

	return keyValueMap
}

func (c *Client) GetDBOneKeyValues() map[string][]string {
	var (
		res         interface{}
		err         error
		keyValueMap map[string][]string
	)

	if res, err = c.DBOne.Do("KEYS", "*").Result(); err != nil {
		log.Println("Error Printing DB One (1): ", err)
	}

	keyValueMap = make(map[string][]string)

	switch keys := res.(type) {
	case []interface{}:
		for _, key := range keys {
			if smembers, err := c.DBOne.Do("SMEMBERS", key).Result(); err != nil {
				log.Println("Error Printing DB One (1): ", err)
			} else {
				keyValueMap[key.(string)] = []string{}
				switch values := smembers.(type) {
				case []interface{}:
					for _, value := range values {
						keyValueMap[key.(string)] = append(keyValueMap[key.(string)], value.(string))
					}
				default:
					fmt.Printf("Cannot iterate over %T\n", smembers)
				}
			}
		}
	default:
		fmt.Printf("Cannot iterate over %T\n", res)
	}

	return keyValueMap
}
